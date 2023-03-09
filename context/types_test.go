package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins/souin/configuration"
	"go.uber.org/zap"
)

func Test_GetContext(t *testing.T) {
	if GetContext() == nil {
		t.Error("The context object must not be nil.")
	}
}

func Test_Context_Init(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := configuration.Configuration{
		DefaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	co := GetContext()

	co.Init(&c)
}

func Test_Context_SetContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := configuration.Configuration{
		DefaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	co := GetContext()

	co.Init(&c)
	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req = co.SetContext(req)
	if req.Context().Value(Key) != "GET-http-domain.com-" {
		t.Errorf("The Key context must be equal to GET-http-domain.com-, %s given.", req.Context().Value(Key))
	}
	if req.Context().Value(GraphQL) != false {
		t.Error("The GraphQL context must be false.")
	}
	if req.Context().Value(SupportedMethod) != nil {
		t.Error("The SupportedMethod context must be nil.")
	}
	if req.Context().Value(HashBody) != "" {
		t.Error("The HashBody context must be an empty string.")
	}
}

func Test_Context_SetBaseContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{
		CacheName: "Dummy",
	}
	c := configuration.Configuration{
		DefaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	co := GetContext()

	co.Init(&c)
	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req = co.SetBaseContext(req)
	if !req.Context().Value(SupportedMethod).(bool) {
		t.Error("The SupportedMethod context must be true.")
	}
	if req.Context().Value(CacheName).(string) != "Dummy" {
		t.Error("The cache name must not be equal to Something.")
	}

	req = httptest.NewRequest(http.MethodPost, "http://domain.com", nil)
	req = co.SetBaseContext(req)
	if req.Context().Value(SupportedMethod).(bool) {
		t.Error("The SupportedMethod context must be false.")
	}
	if req.Context().Value(CacheName).(string) != "Dummy" {
		t.Error("The cache name must not be equal to Something.")
	}
}
