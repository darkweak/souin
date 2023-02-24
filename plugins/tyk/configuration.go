package main

import (
	"bytes"
	"sync"

	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins/souin/agnostic"
)

const (
	configKey       string = "httpcache"
	path            string = "path"
	url             string = "url"
	configurationPK string = "configuration"
)

func parseConfiguration(id string, c map[string]interface{}) *souinInstance {
	c = c[configKey].(map[string]interface{})
	var configuration middleware.BaseConfiguration
	agnostic.ParseConfiguration(&configuration, c)

	s := newInstanceFromConfiguration(configuration)
	definitions[id] = s

	return s
}

func newInstanceFromConfiguration(c middleware.BaseConfiguration) *souinInstance {
	bufPool := &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	ctx := context.GetContext()
	ctx.Init(&c)
	return &souinInstance{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
		bufPool:          bufPool,
		context:          ctx,
	}
}
