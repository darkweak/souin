package main

import (
	"bytes"
	"sync"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/plugins"
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
	var configuration plugins.BaseConfiguration
	agnostic.ParseConfiguration(&configuration, c)

	s := newInstanceFromConfiguration(configuration)
	definitions[id] = s

	return s
}

func newInstanceFromConfiguration(c plugins.BaseConfiguration) *souinInstance {
	s := &souinInstance{
		Configuration: &c,
		Retriever:     plugins.DefaultSouinPluginInitializerFromConfiguration(&c),
		bufPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
		RequestCoalescing: coalescing.Initialize(),
	}
	s.MapHandler = api.GenerateHandlerMap(s.Configuration, s.Retriever.GetTransport())

	return s
}
