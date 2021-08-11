package main

import (
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"io"
	"net/http"
	"net/http/httptest"
	"time"
)

func SouinRequestHandler(rw http.ResponseWriter, r *http.Request) {
	// TODO remove these lines once Tyk patch the
	// ctx.GetDefinition(r)
	def := apiDefinitionRetriever(r.Context())
	currentAPI := ""
	if def != nil {
		currentAPI = def.APIID
	}
	currentInstance := s.configurations[currentAPI]
	r.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	plugins.DefaultSouinPluginCallback(rw, r, currentInstance.Retriever, currentInstance.RequestCoalescing, func(_ http.ResponseWriter, _ *http.Request) error {
		recorder := httptest.NewRecorder()
		var e error

		response := recorder.Result()
		r.Response = response
		response, e = currentInstance.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(r)
		if e != nil {
			return e
		}

		_, e = io.Copy(rw, response.Body)

		return e
	})
}

func init() {
	s.configurations = fromDir("/opt/tyk-gateway/apps")
}

type souinInstance struct {
	RequestCoalescing coalescing.RequestCoalescingInterface
	Retriever         types.RetrieverResponsePropertiesInterface
	Configuration     *Configuration
}

type souinInstances struct {
	configurations map[string]souinInstance
}

// plugin internal state and implementation
var (
	s souinInstances
)

func main() {}
