package main

import (
	"context"
	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/rfc"
	"net/http"
	"strings"
	"time"

	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/plugins"
)

var (
	currentCtx context.Context = nil
)

func getInstanceFromRequest(r *http.Request) *souinInstance {
	if currentCtx == nil {
		currentCtx = r.Context()
	}
	def := apiDefinitionRetriever(currentCtx)
	currentAPI := ""
	if def != nil {
		currentAPI = def.APIID
	}
 	return s.configurations[currentAPI]
}

func SouinResponseHandler(rw http.ResponseWriter, res *http.Response, _ *http.Request) {
	req := res.Request
	req.Response = res
	currentInstance := getInstanceFromRequest(req)
	if b, _ := currentInstance.HandleInternally(req); b {
		// handler(rw, req)
		return
	}

	if !strings.Contains(req.Header.Get("Cache-Control"), "no-cache") {
		retriever := currentInstance.Retriever
		key := rfc.GetCacheKey(req)
		r, _ := rfc.CachedResponse(
			retriever.GetProvider(),
			req,
			key,
			retriever.GetTransport(),
			false,
		)

		if r.Response != nil {
			rh := r.Response.Header
			rfc.HitCache(&rh)
			r.Response.Header = rh
			for _, v := range []string{"Age", "Cache-Status"} {
				h := r.Response.Header.Get(v)
				if h != "" {
					rw.Header().Set(v, h)
				}
			}
			res = r.Response
		} else {
			res, _ = retriever.GetTransport().UpdateCacheEventually(req)
		}
	}

	currentCtx = nil
}

// SouinRequestHandler handle the Tyk request
func SouinRequestHandler(rw http.ResponseWriter, r *http.Request) {
	// TODO remove these lines once Tyk patch the
	// ctx.GetDefinition(r)
	currentInstance := getInstanceFromRequest(r)
	if b, handler := currentInstance.HandleInternally(r); b {
		handler(rw, r)
		return
	}
	r.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	coalescing.ServeResponse(rw, r, currentInstance.Retriever, plugins.DefaultSouinPluginCallback, currentInstance.RequestCoalescing, func(_ http.ResponseWriter, _ *http.Request) error {
		return nil
	})
}

func init() {
	s.configurations = fromDir("/opt/tyk-gateway/apps")
}

type souinInstance struct {
	RequestCoalescing coalescing.RequestCoalescingInterface
	Retriever         types.RetrieverResponsePropertiesInterface
	Configuration     *Configuration
	MapHandler        *api.MapHandler
}

func (s *souinInstance) HandleInternally(r *http.Request) (bool, func(http.ResponseWriter, *http.Request)) {
	if s.MapHandler != nil {
		for k, souinHandler := range *s.MapHandler.Handlers {
			if strings.Contains(r.RequestURI, k) {
				return true, souinHandler
			}
		}
	}

	return false, nil
}

type souinInstances struct {
	configurations map[string]*souinInstance
}

// plugin internal state and implementation
var (
	s souinInstances
)

func main() {}
