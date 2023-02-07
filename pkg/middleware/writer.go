package middleware

import (
	"bytes"
	"net/http"
	"strings"

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
	}
}

// CustomWriter handles the response and provide the way to cache the value
type CustomWriter struct {
	Buf         *bytes.Buffer
	Rw          http.ResponseWriter
	Req         *http.Request
	Headers     http.Header
	headersSent bool
	statusCode  int
	// size        int
}

// Header will write the response headers
func (r *CustomWriter) Header() http.Header {
	if r.headersSent {
		return http.Header{}
	}
	return r.Rw.Header()
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
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
	// r.Response.Header.Set("Content-Length", fmt.Sprint(r.size))
	return len(b), nil
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	r.Headers.Del(rfc.StoredTTLHeader)
	defer r.Buf.Reset()
	// TODO re-enable esi parsing
	// b := esi.Parse(r.Buf.Bytes(), r.Req)
	for h, v := range r.Headers {
		if len(v) > 0 {
			r.Rw.Header().Set(h, strings.Join(v, ", "))
		}
	}

	if !r.headersSent {
		// r.Rw.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
		r.Rw.WriteHeader(r.statusCode)
		r.headersSent = true
	}
	return r.Rw.Write(r.Buf.Bytes())
}
