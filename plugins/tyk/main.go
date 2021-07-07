package main

import (
	"fmt"
	"github.com/TykTechnologies/tyk/ctx"
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
	fmt.Println("Start souin handler")
	fmt.Printf("%+v\n", r)
	fmt.Printf("%+v\n", s)
	session := ctx.GetSession(r)
	fmt.Printf("%+v\n", session)
	fmt.Println("Developer ID:", session.MetaData["tyk_developer_id"])
	fmt.Println("Developer Email:", session.MetaData["tyk_developer_email"])
	apidef := ctx.GetDefinition(r)
	fmt.Println("API name is", apidef.Name)
	//currentAPI := ctx.GetDefinition(r).APIID
	fmt.Printf("===== START =====")
	for i := 1; i < 24; i++ {
		fmt.Printf("%+v\n", r.Context().Value(i))
	}
	fmt.Printf("===== STOP =====")
	//rw.Write([]byte("\n"))
	//rw.Write([]byte(fmt.Sprintf("%+v\n", s.configurations[currentAPI])))
}

func init() {
	s.configurations = fromDir("/opt/tyk-gateway/apps")
}

type souinInstance struct {
	configurations map[string]Configuration
}

// plugin internal state and implementation
var (
	s souinInstance
)

func main() {}

