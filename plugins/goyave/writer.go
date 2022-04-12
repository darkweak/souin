package goyave

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"goyave.dev/goyave/v4"
)

type (
	goyaveWriterDecorator struct {
		goyaveResponse *goyave.Response
		request        *http.Request
		updateCache    func(req *http.Request) (*http.Response, error)
		buf            *bytes.Buffer
		writer         io.Writer
		Response       *http.Response
	}
)

func (g *goyaveWriterDecorator) Send() (int, error) {
	return g.Write(g.buf.Bytes())
}

// Header will write the response headers
func (g *goyaveWriterDecorator) Header() http.Header {
	return g.goyaveResponse.Header()
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
		b, _ = ioutil.ReadAll(g.Response.Body)
	}
	return len(b), nil
}

func (g *goyaveWriterDecorator) PreWrite(b []byte) {
	g.Response.StatusCode = g.goyaveResponse.GetStatus()
	g.buf.Write(b)
	g.Response.Body = ioutil.NopCloser(g.buf)
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
