package main

import (
	"encoding/json"
	"github.com/TykTechnologies/tyk/config"
	"net/http"
)

func SouinRequestHandler(rw http.ResponseWriter, r *http.Request) {
	//r.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
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

	b, _ := json.Marshal(s)
	rw.Write(b)
}

func init() {
	c := fromDir(config.Global().AppPath)
	s.Configuration = &c
}

type souinInstance struct {
	Configuration *Configuration
}

// plugin internal state and implementation
var (
	s *souinInstance
)

func main() {}

