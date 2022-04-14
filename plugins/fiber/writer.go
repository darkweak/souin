package fiber

import (
	"bytes"
	"net/http"

	"github.com/darkweak/souin/plugins"
	"github.com/gofiber/fiber/v2"
)

type (
	fiberWriterDecorator struct {
		*plugins.CustomWriter
	}
	fiberWriter struct {
		h http.Header
		*fiber.Ctx
	}
)

func (f *fiberWriter) Header() http.Header {
	if f.h == nil {
		f.h = http.Header{}
	}
	return f.h
}

func (f *fiberWriter) WriteHeader(code int) {
	f.Ctx.Response().Header.SetStatusCode(code)
}

func (f *fiberWriterDecorator) Send() (int, error) {
	b := new(bytes.Buffer)
	_, _ = b.ReadFrom(f.CustomWriter.Response.Body)
	return b.Len(), nil
}
