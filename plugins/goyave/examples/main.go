package main

import (
	"net/http"

	cache "github.com/darkweak/souin/plugins/goyave"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/config"
)

func main() {
	config.LoadFrom("examples/config.json")
	goyave.Start(func(r *goyave.Router) {
		r.Get("/{p}", func(response *goyave.Response, r *goyave.Request) {
			response.String(http.StatusOK, "Hello, World 👋!")
		}).Middleware(cache.NewHTTPCache(cache.DevDefaultConfiguration).Handle)
	})
}
