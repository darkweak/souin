package webgo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bnkamalesh/webgo/v6"
	"github.com/darkweak/souin/plugins"
)

func Test_NewHTTPCache(t *testing.T) {
	s := NewHTTPCache(DevDefaultConfiguration)
	if s.bufPool == nil {
		t.Error("The bufpool must be set.")
	}
	c := plugins.BaseConfiguration{}
	defer func() {
		if recover() == nil {
			t.Error("The New method must crash if an incomplete configuration is provided.")
		}
	}()
	_ = NewHTTPCache(c)
}

func defaultHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}

func excludedHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, Excluded!"))
}

func getRoutes() []*webgo.Route {
	return []*webgo.Route{
		{
			Name:     "default",
			Method:   http.MethodGet,
			Pattern:  "/:a*",
			Handlers: []http.HandlerFunc{defaultHandler},
		},
	}
}

func prepare() (res *httptest.ResponseRecorder, res2 *httptest.ResponseRecorder, router *webgo.Router) {
	cfg := &webgo.Config{
		Host:         "",
		Port:         "80",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 1 * time.Hour,
	}
	res = httptest.NewRecorder()
	res2 = httptest.NewRecorder()
	httpcache := NewHTTPCache(DevDefaultConfiguration)
	router = webgo.NewRouter(cfg, getRoutes()...)
	router.Use(httpcache.Middleware)
	return
}

func Benchmark_SouinWebgoPlugin_Middleware(b *testing.B) {
	for i := 0; i < b.N; i++ {
		res := httptest.NewRecorder()
		httpcache := NewHTTPCache(DevDefaultConfiguration)
		httpcache.Middleware(res, httptest.NewRequest(http.MethodGet, "/handled"+strconv.Itoa(i), nil), func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Returns something"))
		})
	}
}

func Test_SouinWebgoPlugin_Middleware(t *testing.T) {
	res, res2, router := prepare()
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}
	router.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}

	router.ServeHTTP(res2, req)
	if res2.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res2.Result().Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_SouinWebgoPlugin_Middleware_CannotHandle(t *testing.T) {
	res, res2, router := prepare()
	req := httptest.NewRequest(http.MethodGet, "/not-handled", nil)
	req.Header = http.Header{}
	req.Header.Add("Cache-Control", "no-cache")
	router.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; key=; detail=CANNOT-HANDLE" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}

	router.ServeHTTP(res2, req)
	if res2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; key=; detail=CANNOT-HANDLE" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res2.Result().Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_SouinWebgoPlugin_Middleware_APIHandle(t *testing.T) {
	time.Sleep(DevDefaultConfiguration.DefaultCache.TTL.Duration)
	res, res2, router := prepare()
	req := httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil)
	req.Header = http.Header{}
	router.ServeHTTP(res, req)

	if res.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ := io.ReadAll(res.Result().Body)
	defer res.Result().Body.Close()
	if string(b) != "[]" {
		t.Error("The response body must be an empty array because no request has been stored")
	}
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/handled", nil))
	router.ServeHTTP(res2, httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil))
	if res2.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ = io.ReadAll(res2.Result().Body)
	defer res2.Result().Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 2 {
		t.Error("The system must store 2 items, the fresh and the stale one")
	}
	if payload[0] != "GET-example.com-/handled" || payload[1] != "STALE_GET-example.com-/handled" {
		t.Error("The payload items mismatch from the expectations.")
	}
}
