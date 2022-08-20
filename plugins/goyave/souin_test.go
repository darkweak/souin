package goyave

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/plugins"
	"goyave.dev/goyave/v4"
	"goyave.dev/goyave/v4/config"
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

type HttpCacheMiddlewareTestSuite struct {
	goyave.TestSuite
}

func prepare(suite *HttpCacheMiddlewareTestSuite, request *http.Request) (*SouinGoyaveMiddleware, *goyave.Request) {
	return NewHTTPCache(DevDefaultConfiguration), suite.CreateTestRequest(request)
}

func TestHttpCacheMiddlewareTestSuite(t *testing.T) {
	_ = config.LoadFrom("examples/config.json")
	goyave.RunTest(t, new(HttpCacheMiddlewareTestSuite))
}

func (suite *HttpCacheMiddlewareTestSuite) Test_SouinFiberPlugin_Middleware() {
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}
	httpcache, request := prepare(suite, req)
	res := suite.Middleware(httpcache.Handle, request, func(response *goyave.Response, r *goyave.Request) {
		response.String(http.StatusOK, "Hello, World 👋!")
	})

	b, err := io.ReadAll(res.Body)
	if err != nil {
		suite.T().Error(err)
	}

	if string(b) != "Hello, World 👋!" {
		suite.T().Error("The response body must be equal to Hello, World 👋!.")
	}

	if res.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		suite.T().Error("The response must contain a Cache-Status header with the stored directive.")
	}

	time.Sleep(time.Millisecond)
	res = suite.Middleware(httpcache.Handle, request, func(response *goyave.Response, r *goyave.Request) {})

	b, err = io.ReadAll(res.Body)
	if err != nil {
		suite.T().Error(err)
	}

	if string(b) != "Hello, World 👋!" {
		suite.T().Error("The response body must be equal to Hello, World 👋!.")
	}

	if res.Header.Get("Cache-Status") != "Souin; hit; ttl=4" {
		suite.T().Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res.Header.Get("Age") != "1" {
		suite.T().Error("The response must contain a Age header with the value 1.")
	}
}

func (suite *HttpCacheMiddlewareTestSuite) Test_SouinFiberPlugin_Middleware_CannotHandle() {
	req := httptest.NewRequest(http.MethodGet, "/not-handled", nil)
	req.Header = http.Header{}
	req.Header.Add("Cache-Control", "no-cache")
	httpcache, request := prepare(suite, req)
	res := suite.Middleware(httpcache.Handle, request, func(response *goyave.Response, r *goyave.Request) {
		response.String(http.StatusOK, "Hello, World 👋!")
	})

	b, err := io.ReadAll(res.Body)
	if err != nil {
		suite.T().Error(err)
	}
	if string(b) != "Hello, World 👋!" {
		suite.T().Error("The response body must be equal to Hello, World 👋!.")
	}

	if res.Header.Get("Cache-Status") != "Souin; fwd=uri-miss" {
		suite.T().Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}

	res = suite.Middleware(httpcache.Handle, request, func(response *goyave.Response, r *goyave.Request) {})

	if res.Header.Get("Cache-Status") != "Souin; fwd=uri-miss" {
		suite.T().Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res.Header.Get("Age") != "" {
		suite.T().Error("The response must not contain a Age header.")
	}
}

func (suite *HttpCacheMiddlewareTestSuite) Test_SouinFiberPlugin_Middleware_APIHandle() {
	time.Sleep(DevDefaultConfiguration.DefaultCache.GetTTL() + DevDefaultConfiguration.GetDefaultCache().GetStale())
	req := httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil)
	req.Header = http.Header{}
	httpcache, SouinAPIRequest := prepare(suite, req)
	res := suite.Middleware(httpcache.Handle, SouinAPIRequest, func(response *goyave.Response, r *goyave.Request) {})
	time.Sleep(DevDefaultConfiguration.DefaultCache.GetTTL() + DevDefaultConfiguration.GetDefaultCache().GetStale())

	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(b) != "[]" {
		suite.T().Error("The response body must be an empty array because no request has been stored")
	}
	_ = suite.Middleware(httpcache.Handle, suite.CreateTestRequest(httptest.NewRequest(http.MethodGet, "/handled", nil)), func(response *goyave.Response, r *goyave.Request) {
		response.String(http.StatusOK, "Hello, World 👋!")
	})
	time.Sleep(10*time.Millisecond)
	res = suite.Middleware(httpcache.Handle, SouinAPIRequest, func(response *goyave.Response, r *goyave.Request) {})
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 2 {
		suite.T().Error("The system must store 2 items, the fresh and the stale one")
	}
	if payload[0] != "GET-example.com-/handled" || payload[1] != "STALE_GET-example.com-/handled" {
		suite.T().Error("The payload items mismatch from the expectations.")
	}
}
