package middleware

import (
	"bytes"
	"net/http"
	"strconv"
	"sync"
	"time"

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

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	defer r.handleBuffer(func(b *bytes.Buffer) {
		b.Reset()
	})
	storedLength := r.Header().Get(rfc.StoredLengthHeader)
	if storedLength != "" {
		r.Header().Set("Content-Length", storedLength)
	}

	result := esi.Parse(r.Buf.Bytes(), r.Req)

	r.Header().Del(rfc.StoredLengthHeader)
	r.Header().Del(rfc.StoredTTLHeader)

	// When the client issued a range request, serve it from the fully cached
	// body through the standard library. http.ServeContent implements RFC 7233
	// in full (single/suffix/multipart ranges, If-Range, 416 with Content-Range,
	// Accept-Ranges and CRLF-delimited multipart payloads), which avoids the
	// off-by-one and out-of-bounds issues of a hand-rolled implementation.
	if rangeHeader := r.Headers.Get("Range"); rangeHeader != "" && r.GetStatusCode() == http.StatusOK && !r.headersSent {
		r.Header().Set("Accept-Ranges", "bytes")

		// ServeContent reads Range/If-Range from the request. Build a minimal
		// request carrying only those headers so it doesn't re-evaluate the
		// conditional headers Souin already handled upstream.
		rangeReq := &http.Request{Method: r.Req.Method, Header: http.Header{"Range": {rangeHeader}}}
		if ifRange := r.Req.Header.Get("If-Range"); ifRange != "" {
			rangeReq.Header.Set("If-Range", ifRange)
		}

		// Last-Modified lets ServeContent evaluate a date-based If-Range.
		var modtime time.Time
		if lastModified := r.Header().Get("Last-Modified"); lastModified != "" {
			if parsed, err := http.ParseTime(lastModified); err == nil {
				modtime = parsed
			}
		}

		r.headersSent = true
		// An empty name skips extension-based sniffing so the cached
		// Content-Type is preserved.
		http.ServeContent(r.Rw, rangeReq, "", modtime, bytes.NewReader(result))

		return len(result), nil
	}

	if len(result) != 0 {
		r.Header().Set("Content-Length", strconv.Itoa(len(result)))
	}

	if !r.headersSent {
		r.Rw.WriteHeader(r.GetStatusCode())
		r.headersSent = true
	}

	return r.Rw.Write(result)
}
