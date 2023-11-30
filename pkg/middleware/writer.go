package middleware

import (
	"bytes"
	"net/http"
	"sync"

	"github.com/darkweak/go-esi/esi"
	"github.com/darkweak/souin/pkg/rfc"
	"golang.org/x/exp/maps"
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
		mutex:      &sync.Mutex{},
	}
}

// CustomWriter handles the response and provide the way to cache the value
type CustomWriter struct {
	Buf         *bytes.Buffer
	Rw          http.ResponseWriter
	Req         *http.Request
	Headers     http.Header
	headersSent bool
	mutex       *sync.Mutex
	statusCode  int
	// size        int
}

// Header will write the response headers
func (r *CustomWriter) Header() http.Header {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.headersSent {
		return http.Header{}
	}
	return r.Rw.Header()
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.headersSent {
		return
	}
	r.Headers = r.Rw.Header()
	r.statusCode = code
	// r.headersSent = true
	// r.Rw.WriteHeader(code)
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.Buf.Grow(len(b))
	_, _ = r.Buf.Write(b)

	return len(b), nil
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	contentLength := r.Headers.Get(rfc.StoredLengthHeader)
	if contentLength != "" {
		r.Header().Set("Content-Length", contentLength)
	}
	defer r.Buf.Reset()
	b := esi.Parse(r.Buf.Bytes(), r.Req)
	maps.Copy(r.Rw.Header(), r.Headers)
	r.Header().Del(rfc.StoredLengthHeader)
	r.Header().Del(rfc.StoredTTLHeader)

	if !r.headersSent {
		
		// r.Rw.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
		r.Rw.WriteHeader(r.statusCode)
		r.headersSent = true
	}

	return r.Rw.Write(b)
}
