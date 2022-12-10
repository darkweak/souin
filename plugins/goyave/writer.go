package goyave

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/rfc"
	"github.com/pquerna/cachecontrol/cacheobject"
	"goyave.dev/goyave/v4"
)

type (
	goyaveWriterDecorator struct {
		goyaveResponse *goyave.Response
		request        *http.Request
		updateCache    func(req *http.Request) (*http.Response, error)
		buf            *bytes.Buffer
		writer         io.Writer
		headersSent    bool
		Response       *http.Response
		Req            *http.Request
	}
)

func (r *goyaveWriterDecorator) calculateCacheHeaders() {
	resco, _ := cacheobject.ParseResponseCacheControl(r.goyaveResponse.Header().Get("Cache-Control"))
	if !rfc.CachableStatusCode(r.Response.StatusCode) || resco.NoStore || r.Req.Context().Value(context.RequestCacheControl).(*cacheobject.RequestCacheDirectives).NoStore {
		rfc.MissCache(r.goyaveResponse.Header().Set, r.Req, "CANNOT-STORE")
	}

	if r.goyaveResponse.Header().Get("Cache-Status") == "" {
		r.goyaveResponse.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; stored", r.Req.Context().Value(context.CacheName)))
	}
}

func (g *goyaveWriterDecorator) Send() (int, error) {
	return g.Write(g.buf.Bytes())
}

// Header will write the response headers
func (g *goyaveWriterDecorator) Header() http.Header {
	return g.goyaveResponse.Header()
}

func (r *goyaveWriterDecorator) Flush() {
	for h, v := range r.Response.Header {
		if len(v) > 0 {
			r.goyaveResponse.Header().Set(h, strings.Join(v, ", "))
		}
	}

	if !r.headersSent {
		r.calculateCacheHeaders()
		r.headersSent = true
	}
}

// WriteHeader will write the response headers
func (g *goyaveWriterDecorator) WriteHeader(code int) {
	if g.Response == nil {
		g.Response = &http.Response{
			Header: http.Header{},
		}
	}
	if code != 0 {
		g.Response.StatusCode = code
	}
}

// Write will write the response body
func (g *goyaveWriterDecorator) Write(b []byte) (int, error) {
	g.goyaveResponse.WriteHeader(g.Response.StatusCode)
	g.writer.Write(b)
	if g.Response.Body != nil {
		b, _ = io.ReadAll(g.Response.Body)
	}
	return len(b), nil
}

func (g *goyaveWriterDecorator) PreWrite(b []byte) {
	g.Response.StatusCode = g.goyaveResponse.GetStatus()
	g.buf.Write(b)
	g.Response.Body = io.NopCloser(g.buf)
	g.request.Response = g.Response
	if g.updateCache != nil {
		g.Response, _ = g.updateCache(g.request)
	}
	for k, v := range g.Response.Header {
		g.goyaveResponse.Header().Set(k, v[0])
	}
}

func (g *goyaveWriterDecorator) Close() error {
	return nil
}
