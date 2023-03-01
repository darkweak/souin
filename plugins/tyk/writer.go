package main

import (
	"bytes"
	"net/http"
)

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

func NewCustomWriter(rq *http.Request, rw http.ResponseWriter, b *bytes.Buffer) *CustomWriter {
	return &CustomWriter{
		statusCode: 200,
		Buf:        b,
		Req:        rq,
		Rw:         rw,
		Headers:    http.Header{},
	}
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
