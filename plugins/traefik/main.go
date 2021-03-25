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

func (s *SouinTraefikPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	coalescing.ServeResponse(rw, req, s.Retriever, plugins.DefaultSouinPluginCallback, s.RequestCoalescing, func(w http.ResponseWriter, r *http.Request) error {
		s.next.ServeHTTP(w, r)
		return nil
	})
}
