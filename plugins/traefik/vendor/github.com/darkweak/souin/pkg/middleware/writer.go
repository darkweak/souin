package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
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

func (r *CustomWriter) resetBuffer() {
	r.mutex.Lock()
	r.Buf.Reset()
	r.mutex.Unlock()
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
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.headersSent {
		return
	}
	r.statusCode = code
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.mutex.Lock()
	r.Buf.Grow(len(b))
	_, _ = r.Buf.Write(b)
	r.mutex.Unlock()

	return len(b), nil
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	defer r.resetBuffer()
	storedLength := r.Header().Get(rfc.StoredLengthHeader)
	if storedLength != "" {
		r.Header().Set("Content-Length", storedLength)
	}
	b := esi.Parse(r.Buf.Bytes(), r.Req)
	if len(b) != 0 {
		r.Header().Set("Content-Length", strconv.Itoa(len(b)))
	}
	r.Header().Del(rfc.StoredLengthHeader)
	r.Header().Del(rfc.StoredTTLHeader)

	if !r.headersSent {
		r.Rw.WriteHeader(r.GetStatusCode())
		r.headersSent = true
	}

	return r.Rw.Write(b)
}
