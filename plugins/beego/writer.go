package beego

import (
	"bytes"
	"net/http"
	"strings"

	beegoCtx "github.com/beego/beego/v2/server/web/context"
	"github.com/darkweak/souin/pkg/rfc"
)

// CustomWriter handles the response and provide the way to cache the value
type CustomWriter struct {
	Buf         *bytes.Buffer
	Rw          http.ResponseWriter
	Req         *http.Request
	Headers     http.Header
	headersSent bool
	statusCode  int
	ctx         *beegoCtx.Context
	// size        int
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.ctx.Output.SetStatus(r.statusCode)
	r.Buf.Grow(len(b))
	_, _ = r.Buf.Write(b)
	// r.Response.Header.Set("Content-Length", fmt.Sprint(r.size))
	return len(b), nil
}

// Header will write the response headers
func (r *CustomWriter) Header() http.Header {
	if r.headersSent {
		return http.Header{}
	}
	return r.Rw.Header()
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	r.Headers.Del(rfc.StoredTTLHeader)
	defer r.Buf.Reset()
	// TODO re-enable esi parsing
	// b := esi.Parse(r.Buf.Bytes(), r.Req)
	for h, v := range r.Headers {
		if len(v) > 0 {
			r.ctx.Output.Header(h, strings.Join(v, ", "))
		}
	}

	if !r.headersSent {
		r.Rw.WriteHeader(r.statusCode)
		r.headersSent = true
	}
	return r.Buf.Len(), r.ctx.Output.Body(r.Buf.Bytes())
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
	if r.headersSent {
		return
	}
	r.Headers = r.Rw.Header()
	r.statusCode = code
}
