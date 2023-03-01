package fiber

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type writerOverride struct {
	h http.Header
	*fiber.Response
}

func newWriter(r *fiber.Response) *writerOverride {
	return &writerOverride{
		Response: r,
		h:        http.Header{},
	}
}

func (w *writerOverride) Header() http.Header {
	return w.h
}

func (w *writerOverride) Write(b []byte) (int, error) {
	w.Response.AppendBody(b)
	return len(b), nil
}

func (w *writerOverride) WriteHeader(code int) {
	w.Response.Header.SetStatusCode(code)
}
