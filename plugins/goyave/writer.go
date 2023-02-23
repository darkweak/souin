package goyave

import (
	"io"
	"net/http"
	"strings"

	"github.com/darkweak/souin/pkg/middleware"
)

type baseWriter struct {
	http.ResponseWriter
	subWriter   *middleware.CustomWriter
	headersSent bool
}

func newBaseWriter(w io.Writer, subWriter *middleware.CustomWriter) *baseWriter {
	return &baseWriter{
		ResponseWriter: w.(http.ResponseWriter),
		subWriter:      subWriter,
	}
}

var _ http.ResponseWriter = (*baseWriter)(nil)

func (b *baseWriter) WriteHeader(code int) {
	b.synchronizeHeaders()
}

func (b *baseWriter) synchronizeHeaders() {
	if !b.headersSent {
		for h, v := range b.subWriter.Headers {
			if len(v) > 0 {
				b.ResponseWriter.Header().Set(h, strings.Join(v, ", "))
			}
		}
		b.headersSent = true
	}
}
