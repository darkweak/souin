package goa

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/storages/core"
)

func Test_NewHTTPCache(t *testing.T) {
	s := &SouinGoaMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&DevDefaultConfiguration),
	}
	if s.SouinBaseHandler.Storers == nil || len(s.SouinBaseHandler.Storers) != 1 {
		t.Error("The storer must be set.")
	}
	defer func() {
		if recover() == nil {
			t.Error("The New method must crash if an incomplete configuration is provided.")
		}
	}()
	_ = &SouinGoaMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&middleware.BaseConfiguration{}),
	}
}

type commonHandler struct{}

func (commonHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if strings.Contains(r.RequestURI, "/excluded") {
		_, _ = w.Write([]byte("Hello, Excluded!"))
	} else {
		_, _ = w.Write([]byte("Hello, World!"))
	}
}

func prepare() (res *httptest.ResponseRecorder, res2 *httptest.ResponseRecorder, handler http.Handler) {
	res = httptest.NewRecorder()
	res2 = httptest.NewRecorder()
	return res, res2, NewHTTPCache(DevDefaultConfiguration)(commonHandler{})
}

func Test_SouinGoaPlugin_Middleware(t *testing.T) {
	res, res2, handler := prepare()
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}
	handler.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}

	handler.ServeHTTP(res2, req)
	if res2.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-example.com-/handled; detail=DEFAULT" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res2.Result().Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_SouinGoaPlugin_Middleware_CannotHandle(t *testing.T) {
	res, res2, handler := prepare()
	req := httptest.NewRequest(http.MethodGet, "/not-handled", nil)
	req.Header = http.Header{}
	req.Header.Add("Cache-Control", "no-cache")
	handler.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/not-handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}

	handler.ServeHTTP(res2, req)
	if res2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/not-handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res2.Result().Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_SouinGoaPlugin_Middleware_APIHandle(t *testing.T) {
	core.ResetRegisteredStorages()
	time.Sleep(DevDefaultConfiguration.DefaultCache.GetTTL() + DevDefaultConfiguration.GetDefaultCache().GetStale())
	res, res2, handler := prepare()
	req := httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil)
	req.Header = http.Header{}
	handler.ServeHTTP(res, req)

	if res.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ := io.ReadAll(res.Result().Body)
	_ = res.Result().Body.Close()
	if string(b) != "[]" {
		t.Error("The response body must be an empty array because no request has been stored")
	}
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/handled", nil))
	handler.ServeHTTP(res2, httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil))
	if res2.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ = io.ReadAll(res2.Result().Body)
	_ = res2.Result().Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 1 {
		t.Error("The system must store 1 item")
	}
	if payload[0] != "GET-http-example.com-/handled" {
		t.Error("The payload items mismatch from the expectations.")
	}
}
