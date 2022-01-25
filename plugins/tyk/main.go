package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
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

// SouinResponseHandler stores the response before sent to the client if possible, only returns otherwise
func SouinResponseHandler(rw http.ResponseWriter, res *http.Response, _ *http.Request) {
	req := res.Request
	req.Response = res
	currentInstance := getInstanceFromRequest(req)
	if currentInstance == nil {
		rw.Header().Set("Cache-Status", "Souin; fwd=uri-miss")
		return
	}
	if b, _ := currentInstance.HandleInternally(req); b {
		return
	}
	currentInstance.Retriever.SetMatchedURLFromRequest(req)
	if !plugins.CanHandle(res.Request, currentInstance.Retriever) {
		rw.Header().Set("Cache-Status", "Souin; fwd=uri-miss")
		return
	}

	retriever := currentInstance.Retriever
	key := rfc.GetCacheKey(req)
	r, _ := rfc.CachedResponse(
		retriever.GetProvider(),
		req,
		key,
		retriever.GetTransport(),
		false,
	)

	if r != nil {
		rh := r.Header
		rfc.HitCache(&rh, retriever.GetMatchedURL().TTL.Duration)
		r.Header = rh
		for _, v := range []string{"Age", "Cache-Status"} {
			h := r.Header.Get(v)
			if h != "" {
				rw.Header().Set(v, h)
			}
		}
	} else {
		r, _ = retriever.GetTransport().UpdateCacheEventually(req)
	}

	res = r

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
	if currentInstance == nil || !plugins.CanHandle(r, currentInstance.Retriever) {
		fmt.Println("currentInstance request is null ?", currentInstance, r.URL)
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
