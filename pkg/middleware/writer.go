package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/darkweak/go-esi/esi"
	"github.com/darkweak/souin/pkg/rfc"
)

type SouinWriterInterface interface {
	http.ResponseWriter
	Send() (int, error)
}

var _ SouinWriterInterface = (*CustomWriter)(nil)

func NewCustomWriter(rq *http.Request, rw http.ResponseWriter, b *bytes.Buffer) *CustomWriter {
	return &CustomWriter{
		statusCode: 200,
		Buf:        b,
		Req:        rq,
		Rw:         rw,
		Headers:    http.Header{},
		mutex:      sync.Mutex{},
	}
}

// CustomWriter handles the response and provide the way to cache the value
type CustomWriter struct {
	Buf         *bytes.Buffer
	Rw          http.ResponseWriter
	Req         *http.Request
	Headers     http.Header
	headersSent bool
	mutex       sync.Mutex
	statusCode  int
}

func (r *CustomWriter) handleBuffer(callback func(*bytes.Buffer)) {
	r.mutex.Lock()
	callback(r.Buf)
	r.mutex.Unlock()
}

// Header will write the response headers
func (r *CustomWriter) Header() http.Header {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.headersSent || r.Req.Context().Err() != nil {
		return http.Header{}
	}

	return r.Rw.Header()
}

// GetStatusCode returns the response status code
func (r *CustomWriter) GetStatusCode() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.statusCode
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.headersSent {
		return
	}
	r.statusCode = code
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.handleBuffer(func(actual *bytes.Buffer) {
		actual.Grow(len(b))
		_, _ = actual.Write(b)
	})

	return len(b), nil
}

type rangeValue struct {
	from, to int64
}

const separator = "--SOUIN-HTTP-CACHE-SEPARATOR"

func parseRange(rangeHeaders []string) []rangeValue {
	if len(rangeHeaders) == 0 {
		return nil
	}

	values := make([]rangeValue, len(rangeHeaders))

	for idx, header := range rangeHeaders {
		ranges := strings.Split(header, "-")

		rv := rangeValue{from: -1, to: -1}
		rv.from, _ = strconv.ParseInt(ranges[0], 10, 64)

		if len(ranges) > 1 {
			rv.to, _ = strconv.ParseInt(ranges[1], 10, 64)
			rv.to++
		}

		values[idx] = rv
	}

	return values
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	defer r.handleBuffer(func(b *bytes.Buffer) {
		b.Reset()
	})
	storedLength := r.Header().Get(rfc.StoredLengthHeader)
	if storedLength != "" {
		r.Header().Set("Content-Length", storedLength)
	}

	result := r.Buf.Bytes()

	result = esi.Parse(result, r.Req)

	if r.Headers.Get("Range") != "" {

		var bufStr string
		mimeType := r.Header().Get("Content-Type")

		r.WriteHeader(http.StatusPartialContent)

		rangeHeader := parseRange(strings.Split(strings.TrimPrefix(r.Headers.Get("Range"), "bytes="), ", "))
		bodyBytes := r.Buf.Bytes()

		if len(rangeHeader) == 1 {
			header := rangeHeader[0]
			content := bodyBytes[header.from:]
			if header.to >= 0 {
				content = content[:header.to-header.from]
			}

			r.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", header.from, header.to, r.Buf.Len()))

			result = content
		}

		if len(rangeHeader) > 1 {
			r.Header().Set("Content-Type", "multipart/byteranges; boundary="+separator)

			for _, header := range rangeHeader {

				content := bodyBytes[header.from:]
				if header.to >= 0 {
					content = content[:header.to-header.from]
				}

				bufStr += fmt.Sprintf(`
%s
Content-Type: %s
Content-Range: bytes %d-%d/%d

%s
`, separator, mimeType, header.from, header.to, r.Buf.Len(), content)
			}

			result = []byte(bufStr + separator + "--")
		}
	}

	if len(result) != 0 {
		r.Header().Set("Content-Length", strconv.Itoa(len(result)))
	}

	r.Header().Del(rfc.StoredLengthHeader)
	r.Header().Del(rfc.StoredTTLHeader)

	if !r.headersSent {
		r.Rw.WriteHeader(r.GetStatusCode())
		r.headersSent = true
	}

	return r.Rw.Write(result)
}
