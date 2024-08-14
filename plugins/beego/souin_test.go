package beego

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/beego/beego/v2/core/config/json"
	"github.com/beego/beego/v2/server/web"
	beegoCtx "github.com/beego/beego/v2/server/web/context"
	"github.com/darkweak/souin/pkg/middleware"
)

func Test_NewHTTPCache(t *testing.T) {
	s := NewHTTPCache(DevDefaultConfiguration)
	if s.SouinBaseHandler.Storers == nil || len(s.SouinBaseHandler.Storers) != 1 {
		t.Error("The storer must be set.")
	}
	c := middleware.BaseConfiguration{}
	defer func() {
		if recover() == nil {
			t.Error("The New method must crash if an incomplete configuration is provided.")
		}
	}()
	_ = NewHTTPCache(c)
}

func prepare() (res, res2 *httptest.ResponseRecorder) {
	res = httptest.NewRecorder()
	res2 = httptest.NewRecorder()

	_ = web.LoadAppConfig("json", "beego.json")
	httpcache := NewHTTPCache(DevDefaultConfiguration)
	for _, storer := range httpcache.SouinBaseHandler.Storers {
		_ = storer.Reset()
	}

	web.InsertFilterChain("/*", httpcache.chainHandleFilter)

	web.InsertFilterChain("/*", func(next web.FilterFunc) web.FilterFunc {
		return func(ctx *beegoCtx.Context) {
			_, _ = ctx.ResponseWriter.Write([]byte("hello"))
		}
	})

	web.BeeApp.Handlers.Init()

	return
}

func Test_SouinBeegoPlugin_Middleware(t *testing.T) {
	res, res2 := prepare()
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}
	web.BeeApp.Handlers.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}

	web.BeeApp.Handlers.ServeHTTP(res2, req)
	if res2.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-example.com-/handled; detail=DEFAULT" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res2.Result().Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_SouinBeegoPlugin_Middleware_CannotHandle(t *testing.T) {
	res, res2 := prepare()
	req := httptest.NewRequest(http.MethodGet, "/not-handled", nil)
	req.Header = http.Header{}
	req.Header.Add("Cache-Control", "no-cache")
	web.BeeApp.Handlers.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/not-handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}

	web.BeeApp.Handlers.ServeHTTP(res2, req)
	if res2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-example.com-/not-handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res2.Result().Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_SouinBeegoPlugin_Middleware_APIHandle(t *testing.T) {
	time.Sleep(DevDefaultConfiguration.DefaultCache.GetTTL())
	res, res2 := prepare()
	req := httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil)
	req.Header = http.Header{}
	web.BeeApp.Handlers.ServeHTTP(res, req)

	if res.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ := io.ReadAll(res.Result().Body)
	defer res.Result().Body.Close()
	if string(b) != "[]" {
		t.Error("The response body must be an empty array because no request has been stored")
	}
	web.BeeApp.Handlers.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/handled", nil))
	web.BeeApp.Handlers.ServeHTTP(res2, httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil))
	if res2.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ = io.ReadAll(res2.Result().Body)
	defer res2.Result().Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 1 {
		t.Error("The system must store 1 item, excluding the mapping")
	}
	if payload[0] != "GET-http-example.com-/handled" {
		t.Error("The payload items mismatch from the expectations.")
	}
}
