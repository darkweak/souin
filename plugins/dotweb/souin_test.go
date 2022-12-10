package dotweb

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/plugins"
	"github.com/devfeel/dotweb"
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

func defaultHandler(ctx dotweb.Context) error {
	return ctx.WriteString("Hello, World 👋!")
}

func excludedHandler(ctx dotweb.Context) error {
	return ctx.WriteString("Hello, Excluded!")
}

func prepare() (res *httptest.ResponseRecorder, res2 *httptest.ResponseRecorder, app *dotweb.DotWeb) {
	app = dotweb.New()
	httpcache := NewHTTPCache(DevDefaultConfiguration)
	app.HttpServer.Router().GET("/:p/:n", defaultHandler).Use(httpcache)
	app.HttpServer.Router().GET("/:p", defaultHandler).Use(httpcache)
	res = httptest.NewRecorder()
	res2 = httptest.NewRecorder()
	go app.StartServer(8081)
	time.Sleep(200 * time.Millisecond)

	return
}

func Test_SouinDotwebPlugin_Middleware(t *testing.T) {
	res, res2, router := prepare()
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}

	router.HttpServer.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}

	router.HttpServer.ServeHTTP(res2, req)

	if res2.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res2.Result().Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_SouinDotwebPlugin_Middleware_CannotHandle(t *testing.T) {
	res, res2, router := prepare()
	req := httptest.NewRequest(http.MethodGet, "/not-handled", nil)
	req.Header = http.Header{}
	req.Header.Add("Cache-Control", "no-cache")
	router.HttpServer.ServeHTTP(res, req)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; key=; detail=CANNOT-HANDLE" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}

	router.HttpServer.ServeHTTP(res2, req)
	if res2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; key=; detail=CANNOT-HANDLE" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res2.Result().Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_SouinDotwebPlugin_Middleware_APIHandle(t *testing.T) {
	time.Sleep(DevDefaultConfiguration.DefaultCache.GetTTL() + DevDefaultConfiguration.GetDefaultCache().GetStale())
	res, res2, router := prepare()
	req := httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil)
	req.Header = http.Header{}
	router.HttpServer.ServeHTTP(res, req)

	if res.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ := io.ReadAll(res.Result().Body)
	res.Result().Body.Close()
	if string(b) != "[]" {
		t.Error("The response body must be an empty array because no request has been stored")
	}
	router.HttpServer.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/handled", nil))
	router.HttpServer.ServeHTTP(res2, httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil))
	if res2.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must be in JSON.")
	}
	b, _ = io.ReadAll(res2.Result().Body)
	res2.Result().Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 2 {
		t.Error("The system must store 2 items, the fresh and the stale one")
	}
	if payload[0] != "GET-example.com-/handled" || payload[1] != "STALE_GET-example.com-/handled" {
		t.Error("The payload items mismatch from the expectations.")
	}
}
