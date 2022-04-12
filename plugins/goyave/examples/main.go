package main

import (
	"net/http"

	cache "github.com/darkweak/souin/plugins/goyave"
	"goyave.dev/goyave/v4"
)

func main() {
	goyave.Start(func(r *goyave.Router) {
		r.Get("/{p}", func(response *goyave.Response, r *goyave.Request) {
			response.String(http.StatusOK, "Hello, World 👋!")
		}).Middleware(cache.NewHTTPCache(cache.DevDefaultConfiguration).Handle)
	})
}
