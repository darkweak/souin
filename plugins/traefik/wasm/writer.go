package main

import (
	"net/http"
	"strings"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

var _ http.ResponseWriter = (*writer)(nil)

type writer struct {
	headers http.Header
	api.Response
}

func newWriter(resp api.Response) *writer {
	return &writer{
		headers:  http.Header{},
		Response: resp,
	}
}

func (w *writer) setStatus(status headerStatus) {
	w.headers.Set(REQUEST_HEADER_NAME, string(status))
}
func (w *writer) syncHeaders() {
	for hname, hvalue := range w.headers {
		w.Response.Headers().Set(hname, strings.Join(hvalue, ", "))
	}
}

func (w *writer) Header() http.Header {
	return w.headers
}
func (w *writer) Write(b []byte) (int, error) {
	w.Response.Body().Write(b)

	return len(b), nil
}
func (w *writer) WriteHeader(statusCode int) {
	w.Response.SetStatusCode(uint32(statusCode))
}
