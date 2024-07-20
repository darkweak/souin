package kratos

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
)

var devDefaultConfiguration = middleware.BaseConfiguration{
	API: configurationtypes.API{
		BasePath: "/httpcache_api",
		Prometheus: configurationtypes.APIEndpoint{
			Enable: true,
		},
		Souin: configurationtypes.APIEndpoint{
			BasePath: "/httpcache",
			Enable:   true,
		},
	},
	DefaultCache: &configurationtypes.DefaultCache{
		Regex: configurationtypes.Regex{
			Exclude: "/excluded",
		},
		TTL: configurationtypes.Duration{
			Duration: 5 * time.Second,
		},
	},
	LogLevel: "debug",
}

type next struct{}

func (n *next) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("Hello Kratos!"))
}

var nextFilter = &next{}

func prepare(endpoint string) (req *http.Request, res1 *httptest.ResponseRecorder, res2 *httptest.ResponseRecorder) {
	req = httptest.NewRequest(http.MethodGet, endpoint, nil)
	res1 = httptest.NewRecorder()
	res2 = httptest.NewRecorder()

	return
}

func Test_HttpcacheKratosPlugin_NewHTTPCacheFilterHandler(t *testing.T) {
	if NewHTTPCacheFilter(devDefaultConfiguration) == nil {
		t.Error("The NewHTTPCacheFilter method must return an HTTP Handler.")
	}
}

func Test_HttpcacheKratosPlugin_NewHTTPCacheFilter(t *testing.T) {
	time.Sleep(devDefaultConfiguration.DefaultCache.GetTTL())
	handler := NewHTTPCacheFilter(devDefaultConfiguration)(nextFilter)
	req, res, res2 := prepare("/handled")
	handler.ServeHTTP(res, req)
	rs := res.Result()
	rs.Body.Close()
	if rs.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}
	handler.ServeHTTP(res2, req)
	rs = res2.Result()
	rs.Body.Close()
	if rs.Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-example.com-/handled; detail=DEFAULT" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if rs.Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_HttpcacheKratosPlugin_NewHTTPCacheFilter_Excluded(t *testing.T) {
	time.Sleep(devDefaultConfiguration.DefaultCache.GetTTL())
	handler := NewHTTPCacheFilter(devDefaultConfiguration)(nextFilter)
	req, res, res2 := prepare("/excluded")
	handler.ServeHTTP(res, req)
	rs := res.Result()
	rs.Body.Close()
	if rs.Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=EXCLUDED-REQUEST-URI" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	handler.ServeHTTP(res2, req)
	rs = res2.Result()
	rs.Body.Close()
	if rs.Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=EXCLUDED-REQUEST-URI" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if rs.Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_HttpcacheKratosPlugin_NewHTTPCacheFilter_Mutation(t *testing.T) {
	config := devDefaultConfiguration
	config.DefaultCache.AllowedHTTPVerbs = []string{http.MethodGet, http.MethodPost}
	time.Sleep(config.DefaultCache.GetTTL())
	handler := NewHTTPCacheFilter(config)(nextFilter)
	req, res, res2 := prepare("/handled")
	req.Body = io.NopCloser(bytes.NewBuffer([]byte(`{"query":"mutation":{something mutated}}`)))
	handler.ServeHTTP(res, req)
	rs := res.Result()
	rs.Body.Close()
	if rs.Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=IS-MUTATION-REQUEST" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	req.Body = io.NopCloser(bytes.NewBuffer([]byte(`{"query":"mutation":{something mutated}}`)))
	handler.ServeHTTP(res2, req)
	rs = res2.Result()
	rs.Body.Close()
	if rs.Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=IS-MUTATION-REQUEST" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if rs.Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_HttpcacheKratosPlugin_NewHTTPCacheFilter_API(t *testing.T) {
	time.Sleep(devDefaultConfiguration.DefaultCache.GetTTL())
	handler := NewHTTPCacheFilter(devDefaultConfiguration)(nextFilter)
	req, res, res2 := prepare("/httpcache_api/httpcache")
	handler.ServeHTTP(res, req)
	rs := res.Result()
	defer rs.Body.Close()
	if rs.Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ := io.ReadAll(rs.Body)
	res.Result().Body.Close()
	if string(b) != "[]" {
		t.Error("The response body must be an empty array because no request has been stored")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/handled", nil)
	handler.ServeHTTP(res2, req2)
	rs = res2.Result()
	rs.Body.Close()
	if rs.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}
	res3 := httptest.NewRecorder()
	handler.ServeHTTP(res3, req)
	rs = res3.Result()
	rs.Body.Close()
	if rs.Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ = io.ReadAll(rs.Body)
	rs.Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 1 {
		t.Error("The system must store 1 item, except the mapping")
	}
	if payload[0] != "GET-http-example.com-/handled" {
		t.Error("The payload items mismatch from the expectations.")
	}
}
