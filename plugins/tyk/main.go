package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/TykTechnologies/tyk/ctx"
	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
)

var definitions map[string]*souinInstance = make(map[string]*souinInstance)

func getInstanceFromRequest(r *http.Request) (s *souinInstance) {
	def := ctx.GetDefinition(r)
	var found bool
	if s, found = definitions[def.APIID]; !found {
		s = parseConfiguration(def.APIID, def.ConfigData)
	}

	return s
}

// SouinResponseHandler stores the response before sent to the client if possible, only returns otherwise
func SouinResponseHandler(rw http.ResponseWriter, res *http.Response, rq *http.Request) {
	res.Request.URL.Host = rq.Host
	res.Request.Host = rq.Host
	res.Request.URL.Path = rq.RequestURI
	res.Request.RequestURI = rq.RequestURI
	req := res.Request
	req.Response = res
	s := getInstanceFromRequest(req)
	req = s.Retriever.GetContext().SetContext(s.Retriever.GetContext().SetBaseContext(req))
	res, _ = s.Retriever.GetTransport().UpdateCacheEventually(req)
}

// SouinRequestHandler handle the Tyk request
func SouinRequestHandler(rw http.ResponseWriter, r *http.Request) {
	s := getInstanceFromRequest(r)
	req := s.Retriever.GetContext().SetBaseContext(r)
	if b, handler := s.HandleInternally(req); b {
		handler(rw, req)

		return
	}

	if !plugins.CanHandle(req, s.Retriever) {
		rfc.MissCache(rw.Header().Set, req)

		return
	}

	req = s.Retriever.GetContext().SetContext(req)
	if plugins.HasMutation(req, rw) {
		return
	}
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))

	_ = plugins.DefaultSouinPluginCallback(rw, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
		return nil
	})
}

type souinInstance struct {
	RequestCoalescing coalescing.RequestCoalescingInterface
	Retriever         types.RetrieverResponsePropertiesInterface
	Configuration     *plugins.BaseConfiguration
	bufPool           *sync.Pool
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

func init() {
	fmt.Println(`message="Souin configuration is now loaded."`)
}

func main() {}
