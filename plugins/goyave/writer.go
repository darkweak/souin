package goyave

import (
	"io"
	"net/http"
	"strings"

	"goyave.dev/goyave/v4"
)

type baseResponseWriter struct {
	Response    *goyave.Response
	writer      io.Writer
	h           http.Header
	headersSent bool
}

var _ http.ResponseWriter = (*baseResponseWriter)(nil)

func newBaseResponseWriter(r *goyave.Response) *baseResponseWriter {
	return &baseResponseWriter{
		Response: r,
		writer:   r.Writer(),
		h:        http.Header{},
	}
}

func (b *baseResponseWriter) synchronizeHeaders() {
	if !b.headersSent {
		for h, v := range b.h {
			if len(v) > 0 {
				b.Response.Header().Set(h, strings.Join(v, ", "))
			}
		}
		b.headersSent = true
	}
}

func (b *baseResponseWriter) Write(data []byte) (int, error) {
	b.synchronizeHeaders()
	return b.Response.Write(data)
}

func (b *baseResponseWriter) Header() http.Header {
	b.synchronizeHeaders()
	return b.Response.Header()
}

func (b *baseResponseWriter) WriteHeader(code int) {
	b.synchronizeHeaders()
}

type baseWriter struct {
	writer io.Writer
	h      http.Header
}

func newBaseWriter(w io.Writer, h http.Header) *baseWriter {
	return &baseWriter{
		writer: w,
		h:      h,
	}
}

var _ http.ResponseWriter = (*baseWriter)(nil)

func (b *baseWriter) Write(data []byte) (int, error) {
	return b.writer.Write(data)
}

func (b *baseWriter) Header() http.Header {
	return b.h
}

func (b *baseWriter) WriteHeader(code int) {}
