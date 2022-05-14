package main

import (
	cache "github.com/darkweak/souin/plugins/dotweb"
	"github.com/devfeel/dotweb"
)

func main() {
	app := dotweb.New()

	// Use the Souin default configuration
	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	app.HttpServer.GET("/:p/:n", func(ctx dotweb.Context) error {
		return ctx.WriteString("Hello, World 👋!")
	}).Use(httpcache)
	app.HttpServer.GET("/:p", func(ctx dotweb.Context) error {
		return ctx.WriteString("Hello, World 👋!")
	}).Use(httpcache)

	_ = app.StartServer(80)
}
