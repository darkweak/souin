package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins/souin/configuration"
	"go.uber.org/zap"
)

func Test_CacheContext_SetupContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := configuration.Configuration{
		DefaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	ctx := cacheContext{}

	ctx.SetupContext(&c)
	if ctx.cacheName != "Souin" {
		t.Error("The context must be equal to Souin.")
	}

	c.DefaultCache.CacheName = "Something"
	ctx.SetupContext(&c)
	if ctx.cacheName != "Something" {
		t.Error("The context must be equal to Something.")
	}
}

func Test_CacheContext_SetContext(t *testing.T) {
	ctx := cacheContext{cacheName: "Something"}

	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req.Body = nil
	req = ctx.SetContext(req)
	if req.Context().Value(CacheName).(string) != "Something" {
		t.Error("The cache name must not be equal to Something.")
	}
}
