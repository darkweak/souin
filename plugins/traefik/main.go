package traefik

import (
	"context"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/plugins"
	"net/http"
)

// SouinTraefikPlugin declaration.
type SouinTraefikPlugin struct {
	next http.Handler
	name string
	plugins.SouinBasePlugin
}

// New create Souin instance.
func New(_ context.Context, next http.Handler, config *Configuration, name string) (http.Handler, error) {
	s := &SouinTraefikPlugin{
		name: name,
		next: next,
	}
	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(config)
	s.RequestCoalescing = coalescing.Initialize()
	return s, nil
}

func (e *SouinTraefikPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	coalescing.ServeResponse(rw, req, e.Retriever, plugins.DefaultSouinPluginCallback, e.RequestCoalescing)
	e.next.ServeHTTP(rw, req)
}
