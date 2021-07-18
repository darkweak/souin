package main

import (
	"fmt"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/types"
	"net/http"
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
	r.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	//plugins.DefaultSouinPluginCallback(rw, r, s.Retriever, s.RequestCoalescing, func(_ http.ResponseWriter, _ *http.Request) error {
	//	recorder := httptest.NewRecorder()
	//	var e error
	//
	//	response := recorder.Result()
	//	r.Response = response
	//	response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(r)
	//	if e != nil {
	//		return e
	//	}
	//
	//	_, e = io.Copy(rw, response.Body)
	//
	//	return e
	//})
	fmt.Println("Start souin handler")

	rw.Write([]byte(fmt.Sprintf("%+v\n", s.configurations[currentAPI])))
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
