package httpcache

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
)

func setupEngine(path, content string) *route.Engine {
	engine := server.Default().Engine

	engine.Use(NewHTTPCache(DevDefaultConfiguration))
	engine.GET(path, func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, content)
	})

	return engine
}

func TestGetDefaultRequest(t *testing.T) {
	engine := setupEngine("/default", "Hello default!")

	rr1 := ut.PerformRequest(engine, http.MethodGet, "http://domain.com/default", nil)
	if rr1.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-domain.com-/default" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr1.Result().Header.Get("Cache-Status"))
	}
	if rr1.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr1.Result().Body()) != "Hello default!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr1.Result().Body()))
	}

	rr2 := ut.PerformRequest(engine, http.MethodGet, "http://domain.com/default", nil)
	if rr2.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-domain.com-/default" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr2.Result().Header.Get("Cache-Status"))
	}
	if rr2.Result().Header.Get("Age") != "1" {
		t.Errorf("The Age header response must be equal to 1, %s given.", rr2.Result().Header.Get("Age"))
	}
	if string(rr2.Result().Body()) != "Hello default!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr2.Result().Body()))
	}

	// no-cache
	rr1 = ut.PerformRequest(engine, http.MethodGet, "http://domain.com/default", nil, ut.Header{
		Key:   "Cache-Control",
		Value: "no-cache",
	})
	if rr1.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-domain.com-/default" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr1.Result().Header.Get("Cache-Status"))
	}
	if rr1.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr1.Result().Body()) != "Hello default!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr1.Result().Body()))
	}

	rr2 = ut.PerformRequest(engine, http.MethodGet, "http://domain.com/default", nil, ut.Header{
		Key:   "Cache-Control",
		Value: "no-cache",
	})
	if rr2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-http-domain.com-/default" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr2.Result().Header.Get("Cache-Status"))
	}
	if rr2.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr2.Result().Body()) != "Hello default!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr2.Result().Body()))
	}

	// no-store
	rr1 = ut.PerformRequest(engine, http.MethodGet, "http://domain.com/default", nil, ut.Header{
		Key:   "Cache-Control",
		Value: "no-store",
	})
	if rr1.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-http-domain.com-/default" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr1.Result().Header.Get("Cache-Status"))
	}
	if rr1.Result().Header.Get("Age") != "1" {
		t.Errorf("The Age header response must be equal to 1, %s given.", rr2.Result().Header.Get("Age"))
	}
	if string(rr1.Result().Body()) != "Hello default!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr1.Result().Body()))
	}

	time.Sleep(5 * time.Second)

	rr2 = ut.PerformRequest(engine, http.MethodGet, "http://domain.com/default", nil, ut.Header{
		Key:   "Cache-Control",
		Value: "no-store",
	})
	if rr2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; detail=NO-STORE-DIRECTIVE; key=GET-http-domain.com-/default" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr2.Result().Header.Get("Cache-Status"))
	}
	if rr2.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr2.Result().Body()) != "Hello default!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr2.Result().Body()))
	}
}

func TestGetExcludedRequest(t *testing.T) {
	engine := setupEngine("/excluded", "Hello excluded!")

	rr1 := ut.PerformRequest(engine, http.MethodGet, "http://domain.com/excluded", nil)
	if rr1.Result().Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=EXCLUDED-REQUEST-URI" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr1.Result().Header.Get("Cache-Status"))
	}
	if rr1.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr1.Result().Body()) != "Hello excluded!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr1.Result().Body()))
	}

	rr2 := ut.PerformRequest(engine, http.MethodGet, "http://domain.com/excluded", nil)
	if rr2.Result().Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=EXCLUDED-REQUEST-URI" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr2.Result().Header.Get("Cache-Status"))
	}
	if rr2.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr2.Result().Body()) != "Hello excluded!" {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr2.Result().Body()))
	}
}

func validateStoreAllowedMethods(t *testing.T, engine *route.Engine, method string) {
	rr1 := ut.PerformRequest(engine, method, "http://domain.com/allowed_methods", nil)
	if rr1.Result().Header.Get("Cache-Status") != fmt.Sprintf("Souin; fwd=uri-miss; stored; key=%s-http-domain.com-/allowed_methods", method) {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr1.Result().Header.Get("Cache-Status"))
	}
	if rr1.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr1.Result().Body()) != fmt.Sprintf("Hello %s!", method) {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr1.Result().Body()))
	}

	rr2 := ut.PerformRequest(engine, method, "http://domain.com/allowed_methods", nil)
	if rr2.Result().Header.Get("Cache-Status") != fmt.Sprintf("Souin; hit; ttl=4; key=%s-http-domain.com-/allowed_methods", method) {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr2.Result().Header.Get("Cache-Status"))
	}
	if rr2.Result().Header.Get("Age") != "1" {
		t.Errorf("The Age header response must be equal to 1, %s given.", rr2.Result().Header.Get("Age"))
	}
	if string(rr2.Result().Body()) != fmt.Sprintf("Hello %s!", method) {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr2.Result().Body()))
	}
}

func TestAllowedMethodsRequest(t *testing.T) {
	engine := server.Default().Engine

	c := DevDefaultConfiguration
	c.DefaultCache.AllowedHTTPVerbs = []string{http.MethodGet, http.MethodPost, http.MethodHead}
	engine.Use(NewHTTPCache(c))
	engine.Any("/allowed_methods", func(ctx context.Context, c *app.RequestContext) {
		c.String(consts.StatusOK, "Hello "+string(c.Request.Method())+"!")
	})

	validateStoreAllowedMethods(t, engine, http.MethodGet)
	validateStoreAllowedMethods(t, engine, http.MethodPost)

	rr1 := ut.PerformRequest(engine, http.MethodPut, "http://domain.com/allowed_methods", nil)
	if rr1.Result().Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=UNSUPPORTED-METHOD" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr1.Result().Header.Get("Cache-Status"))
	}
	if rr1.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr1.Result().Body()) != fmt.Sprintf("Hello %s!", http.MethodPut) {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr1.Result().Body()))
	}

	rr2 := ut.PerformRequest(engine, http.MethodPut, "http://domain.com/allowed_methods", nil)
	if rr2.Result().Header.Get("Cache-Status") != "Souin; fwd=bypass; detail=UNSUPPORTED-METHOD" {
		t.Errorf("The Cache-Status header response mismatched the expectations, %s given.", rr2.Result().Header.Get("Cache-Status"))
	}
	if rr2.Result().Header.Get("Age") != "" {
		t.Error("The Age header response must be empty.")
	}
	if string(rr2.Result().Body()) != fmt.Sprintf("Hello %s!", http.MethodPut) {
		t.Errorf("The Body response mismatched the expectations, %s given.", string(rr2.Result().Body()))
	}
}
