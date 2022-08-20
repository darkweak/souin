package fiber

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/plugins"
	"github.com/gofiber/fiber/v2"
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

func prepare() *fiber.App {
	app := fiber.New()

	app.Use(NewHTTPCache(DevDefaultConfiguration).Handle)
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})

	return app
}

func Test_SouinFiberPlugin_Middleware(t *testing.T) {
	app := prepare()
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}

	res, err := app.Test(req)

	if err != nil {
		t.Error(err)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}

	if string(b) != "Hello, World 👋!" {
		t.Error("The response body must be equal to Hello, World 👋!.")
	}

	if res.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}

	time.Sleep(time.Millisecond)
	res, _ = app.Test(req)
	if err != nil {
		t.Error(err)
	}

	if res.Header.Get("Cache-Status") != "Souin; hit; ttl=4" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res.Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_SouinFiberPlugin_Middleware_CannotHandle(t *testing.T) {
	app := prepare()
	req := httptest.NewRequest(http.MethodGet, "/not-handled", nil)
	req.Header = http.Header{}
	req.Header.Add("Cache-Control", "no-cache")

	res, err := app.Test(req)

	if err != nil {
		t.Error(err)
	}

	_, err = io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}

	if res.Header.Get("Cache-Status") != "Souin; fwd=uri-miss" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}

	res, _ = app.Test(req)
	if err != nil {
		t.Error(err)
	}

	if res.Header.Get("Cache-Status") != "Souin; fwd=uri-miss" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res.Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}

func Test_SouinFiberPlugin_Middleware_APIHandle(t *testing.T) {
	time.Sleep(DevDefaultConfiguration.DefaultCache.GetTTL() + DevDefaultConfiguration.GetDefaultCache().GetStale())
	app := prepare()
	req := httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil)
	req.Header = http.Header{}

	res, err := app.Test(req)
	if err != nil {
		t.Error(err)
	}

	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if string(b) != "[]" {
		t.Error("The response body must be an empty array because no request has been stored")
	}
	_, err = app.Test(httptest.NewRequest(http.MethodGet, "/handled", nil))
	if err != nil {
		t.Error(err)
	}
	time.Sleep(10*time.Millisecond)
	res, err = app.Test(httptest.NewRequest(http.MethodGet, "/souin-api/souin", nil))
	if err != nil {
		t.Error(err)
	}
	b, _ = io.ReadAll(res.Body)
	res.Body.Close()
	var payload []string
	_ = json.Unmarshal(b, &payload)
	if len(payload) != 2 {
		t.Error("The system must store 2 items, the fresh and the stale one")
	}
	if payload[0] != "GET-example.com-/handled" || payload[1] != "STALE_GET-example.com-/handled" {
		t.Error("The payload items mismatch from the expectations.")
	}
}
