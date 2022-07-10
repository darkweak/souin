package main

import (
	"fmt"
	"net/http"

	httpcache "github.com/darkweak/souin/plugins/kratos"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	kratos_http "github.com/go-kratos/kratos/v2/transport/http"
	"gopkg.in/yaml.v3"
)

func main() {
	c := config.New(
		config.WithSource(file.NewSource("examples/configuration.yml")),
		config.WithDecoder(func(kv *config.KeyValue, v map[string]interface{}) error {
			return yaml.Unmarshal(kv.Value, v)
		}),
	)
	if err := c.Load(); err != nil {
		panic(err)
	}

	server := kratos_http.NewServer(
		kratos_http.Filter(
			httpcache.NewHTTPCacheFilter(httpcache.ParseConfiguration(c)),
		),
	)

	r := server.Route("")

	r.GET("/{p}", func(ctx kratos_http.Context) error {
		ctx.Response().WriteHeader(http.StatusOK)
		ctx.Response().Write([]byte("Hello Kratos!"))
		return nil
	})

	fmt.Println(server.Server.ListenAndServe())
}
