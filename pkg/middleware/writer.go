package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/darkweak/go-esi/esi"
	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/pkg/rfc"
)

type SouinWriterInterface interface {
	http.ResponseWriter
	Send() (int, error)
}

var _ SouinWriterInterface = (*CustomWriter)(nil)

func NewCustomWriter(
	rq *http.Request,
	rw http.ResponseWriter,
	b *bytes.Buffer,
	maxSize int,
	earlyHintStore func(http.Header),
) *CustomWriter {
	return &CustomWriter{
		statusCode:     200,
		Buf:            b,
		Req:            rq,
		Rw:             rw,
		Headers:        http.Header{},
		mutex:          sync.Mutex{},
		maxSize:        maxSize,
		maxSizeReached: false,
		earlyHintStore: earlyHintStore,
	}
}

// CustomWriter handles the response and provide the way to cache the value
type CustomWriter struct {
	Buf            *bytes.Buffer
	Rw             http.ResponseWriter
	Req            *http.Request
	Headers        http.Header
	headersSent    bool
	mutex          sync.Mutex
	statusCode     int
	maxSize        int
	maxSizeReached bool

	earlyHintStore func(http.Header)
}

func (r *CustomWriter) resetBuffer() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.Buf.Reset()
}

func (r *CustomWriter) copyToBuffer(src io.Reader) (int64, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return io.Copy(r.Buf, src)
}

func (r *CustomWriter) resetAndCopyToBuffer(src io.Reader) (int64, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.Buf.Reset()

	return io.Copy(r.Buf, src)
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
	defer func(h http.Header) {
		r.mutex.Unlock()

		if code == http.StatusEarlyHints {
			r.earlyHintStore(h)
		}
	}(r.Header())

	if r.headersSent {
		return
	}

	r.mutex.Lock()

	r.statusCode = code
	if code == http.StatusEarlyHints {
		r.Rw.WriteHeader(code)
	}
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.maxSizeReached {
		return r.Rw.Write(b)
	}

	if r.maxSize > 0 && (r.Buf.Len()+len(b)) > r.maxSize {
		r.maxSizeReached = true

		if !r.headersSent && r.Req.Context().Err() == nil {
			r.Rw.Header().Set(
				"Cache-Status",
				fmt.Sprintf(
					"%s; fwd=uri-miss; detail=UPSTREAM-RESPONSE-TOO-LARGE; key=%s",
					r.Req.Context().Value(context.CacheName),
					rfc.GetCacheKeyFromCtx(r.Req.Context()),
				),
			)
		}

		_, _ = r.Rw.Write(r.Buf.Bytes())

		r.Buf.Reset()
	}

	r.Buf.Grow(len(b))
	_, _ = r.Buf.Write(b)

	return len(b), nil
}

type rangeValue struct {
	from, to int64
}

const separator = "--SOUIN-HTTP-CACHE-SEPARATOR"

func parseRange(rangeHeaders []string, contentRange string) ([]rangeValue, rangeValue, int64) {
	if len(rangeHeaders) == 0 {
		return nil, rangeValue{}, -1
	}

	crv := rangeValue{from: 0, to: 0}
	var total int64 = -1
	if contentRange != "" {
		crVal := strings.Split(strings.TrimPrefix(contentRange, "bytes "), "/")
		total, _ = strconv.ParseInt(crVal[1], 10, 64)
		total--

		crSplit := strings.Split(crVal[0], "-")
		crv.from, _ = strconv.ParseInt(crSplit[0], 10, 64)
		crv.to, _ = strconv.ParseInt(crSplit[1], 10, 64)
	}

	values := make([]rangeValue, len(rangeHeaders))

	for idx, header := range rangeHeaders {
		ranges := strings.Split(header, "-")
		rv := rangeValue{from: -1, to: total}

		// e.g. Range: -5
		if len(ranges) == 2 && ranges[0] == "" {
			ranges[0] = "-" + ranges[1]
			from, _ := strconv.ParseInt(ranges[0], 10, 64)
			rv.from = total + from

			values[idx] = rv

			continue
		}

		rv.from, _ = strconv.ParseInt(ranges[0], 10, 64)

		if ranges[1] != "" {
			rv.to, _ = strconv.ParseInt(ranges[1], 10, 64)
			rv.to++
		}

		values[idx] = rv
	}

	return values, crv, total + 1
}

// Push implements http.Pusher
func (r *CustomWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.Rw.(http.Pusher)
	if !ok {
		return fmt.Errorf("ResponseWriter does not implement http.Pusher")
	}

	return pusher.Push(target, opts)
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	defer r.resetBuffer()

	storedLength := r.Header().Get(rfc.StoredLengthHeader)
	if storedLength != "" {
		r.Header().Set("Content-Length", storedLength)
	}

	result := r.Buf.Bytes()

	result = esi.Parse(result, r.Req)

	if r.Headers.Get("Range") != "" {

		bufStr := new(strings.Builder)
		mimeType := r.Header().Get("Content-Type")

		r.WriteHeader(http.StatusPartialContent)

		rangeHeader, contentRangeValue, total := parseRange(
			strings.Split(strings.TrimPrefix(r.Headers.Get("Range"), "bytes="), ", "),
			r.Header().Get("Content-Range"),
		)
		bodyBytes := r.Buf.Bytes()
		bufLen := int64(r.Buf.Len())
		if total > 0 {
			bufLen = total
		}

		if len(rangeHeader) == 1 {
			header := rangeHeader[0]
			internalFrom := (header.from - contentRangeValue.from) % bufLen
			internalTo := (header.to - contentRangeValue.from) % bufLen

			content := bodyBytes[internalFrom:]

			r.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", contentRangeValue.from, contentRangeValue.to, bufLen))

			if internalTo >= 0 {
				content = content[:internalTo-internalFrom]
				r.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", header.from, header.to, bufLen))
			}

			result = content
		}

		if len(rangeHeader) > 1 {
			r.Header().Set("Content-Type", "multipart/byteranges; boundary="+separator)

			for _, header := range rangeHeader {

				content := bodyBytes[header.from:]
				if header.to >= 0 {
					content = content[:header.to-header.from]
				}

				bufStr.WriteString(fmt.Sprintf(`
%s
Content-Type: %s
Content-Range: bytes %d-%d/%d

%s
`, separator, mimeType, header.from, header.to, r.Buf.Len(), content))
			}

			bufStr.WriteString(separator + "--")
			result = []byte(bufStr.String())
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

func newSWRRW(closer io.ReadCloser) http.ResponseWriter {
	return &swrResponseWriter{
		body:       closer,
		headers:    http.Header{},
		statusCode: 0,
	}
}

type swrResponseWriter struct {
	body       io.ReadCloser
	headers    http.Header
	statusCode int
}

func (r swrResponseWriter) Header() http.Header {
	return r.headers
}

func (r swrResponseWriter) WriteHeader(code int) {
	r.statusCode = code
}

func (r swrResponseWriter) Write(b []byte) (int, error) {
	return r.body.Read(b)
}
