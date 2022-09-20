package fiber

import (
	"net/http"

	"github.com/darkweak/souin/plugins"
	"github.com/gofiber/fiber/v2"
)

type (
	fiberWriterDecorator struct {
		*plugins.CustomWriter
	}
	nopWriter struct {
		h http.Header
		*fiber.Ctx
	}
	fiberWriter struct {
		h http.Header
		*fiber.Ctx
	}
)

func (f *nopWriter) Header() http.Header {
	if f.h == nil {
		f.h = http.Header{}
	}
	return f.h
}

func (f *fiberWriter) Header() http.Header {
	if f.h == nil {
		f.h = http.Header{}
	}
	return f.h
}

func (f *fiberWriter) WriteHeader(code int) {
	f.Ctx.Response().Header.SetStatusCode(code)
}

func (f *nopWriter) WriteHeader(code int) {
	f.Ctx.Response().Header.SetStatusCode(code)
}

func (f *nopWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (f *fiberWriter) Write(b []byte) (int, error) {
	return f.Ctx.Write(b)
}

func (f *fiberWriterDecorator) Send() (int, error) {
	return f.CustomWriter.Send()
}
