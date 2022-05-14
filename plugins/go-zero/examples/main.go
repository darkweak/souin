package main

import (
	"flag"

	cache "github.com/darkweak/souin/plugins/go-zero"

	"github.com/darkweak/souin/plugins/go-zero/examples/internal/config"
	"github.com/darkweak/souin/plugins/go-zero/examples/internal/handler"
	"github.com/darkweak/souin/plugins/go-zero/examples/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "sample.yml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	server.Use(httpcache.Handle)

	handler.RegisterHandlers(server, ctx)

	server.Start()
}
