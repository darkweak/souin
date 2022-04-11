package main

import (
	"net/http"

	cache "github.com/darkweak/souin/plugins/goyave"
	"goyave.dev/goyave/v4"
)

func main() {
	goyave.Start(func(r *goyave.Router) {
		r.GlobalMiddleware(cache.NewHTTPCache(cache.DevDefaultConfiguration).Handle)
		r.Route("GET", "/*", func(response *goyave.Response, r *goyave.Request) {
			response.String(http.StatusOK, "Hello, World 👋!")
		})
	})
}
