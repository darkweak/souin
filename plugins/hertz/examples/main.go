package main

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	httpcache "github.com/darkweak/souin/plugins/hertz"
)

func main() {
	h := server.Default(server.WithHostPorts(":80"))

	h.Use(httpcache.NewHTTPCache(httpcache.DevDefaultConfiguration))

	h.GET("/*path", func(c context.Context, ctx *app.RequestContext) {
		ctx.String(http.StatusOK, "Hello Hertz!")
	})

	h.Spin()
}
