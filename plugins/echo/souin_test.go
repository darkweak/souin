package souin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/labstack/echo/v4"
)

func Test_New(t *testing.T) {
	s := NewMiddleware(DevDefaultConfiguration)
	if s.Storer == nil {
		t.Error("The storer must be set.")
	}
	c := middleware.BaseConfiguration{}
	defer func() {
		if recover() == nil {
			t.Error("The New method must crash if an incomplete configuration is provided.")
		}
	}()
	NewMiddleware(c)
}

func Test_SouinEchoPlugin_Process(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}
	res := httptest.NewRecorder()
	res2 := httptest.NewRecorder()
	s := NewMiddleware(DevDefaultConfiguration)

	e := echo.New()
	c := e.NewContext(req, res)
	c2 := e.NewContext(req, res2)
	handler := func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "test")
	}

	if err := s.Process(handler)(c); err != nil {
		t.Error("No error must be thrown if everything is good.")
	}
	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}
	if err := s.Process(handler)(c2); err != nil {
		t.Error("No error must be thrown on the second request if everything is good.")
	}
	if res2.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res2.Result().Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_SouinEchoPlugin_Process_CannotHandle(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/not-handled", nil)
	req.Header = http.Header{}
	req.Header.Add("Cache-Control", "no-cache")
	res := httptest.NewRecorder()
	res2 := httptest.NewRecorder()
	s := NewMiddleware(DevDefaultConfiguration)

	e := echo.New()
	c := e.NewContext(req, res)
	c2 := e.NewContext(req, res2)
	handler := func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "test")
	}

	if err := s.Process(handler)(c); err != nil {
		t.Error("No error must be thrown if everything is good.")
	}
	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-example.com-/not-handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if err := s.Process(handler)(c2); err != nil {
		t.Error("No error must be thrown on the second request if everything is good.")
	}
	if res2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-example.com-/not-handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res2.Result().Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_SouinEchoPlugin_Process_APIHandle(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil)
	req.Header = http.Header{}
	res := httptest.NewRecorder()
	dc := DevDefaultConfiguration
	dc.DefaultCache.Nuts = configurationtypes.CacheProvider{
		Path: "/tmp/souin" + time.Now().UTC().String(),
	}
	s := NewMiddleware(dc)

	e := echo.New()
	c := e.NewContext(req, res)
	handler := func(ctx echo.Context) error {
		return ctx.String(http.StatusOK, "test")
	}

	if err := s.Process(handler)(c); err != nil {
		t.Error("No error must be thrown if everything is good.")
	}
	if res.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must contain be in JSON.")
	}
	b, _ := io.ReadAll(res.Result().Body)
	defer res.Result().Body.Close()
	if string(b) != "[]" {
		t.Error("The response body must be an empty array because no request has been stored")
	}
	rs := httptest.NewRequest(http.MethodGet, "/handled", nil)
	_ = s.Process(handler)(e.NewContext(rs, res))
	res2 := httptest.NewRecorder()
	if err := s.Process(handler)(e.NewContext(req, res2)); err != nil {
		t.Error("No error must be thrown if everything is good.")
	}
	if res.Result().Header.Get("Content-Type") != "application/json" {
		t.Error("The response must contain be in JSON.")
	}
	b, _ = io.ReadAll(res2.Result().Body)
	defer res.Result().Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 2 {
		t.Error("The system must store 2 items, the fresh and the stale one")
	}
	if payload[0] != "GET-example.com-/handled" || payload[1] != "STALE_GET-example.com-/handled" {
		t.Error("The payload items mismatch from the expectations.")
	}
}
