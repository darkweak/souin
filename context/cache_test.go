package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

type testConfiguration struct {
	defaultCache *configurationtypes.DefaultCache
	cacheKeys    configurationtypes.CacheKeys
}

func (*testConfiguration) GetUrls() map[string]configurationtypes.URL {
	return nil
}
func (*testConfiguration) GetPluginName() string {
	return ""
}
func (t *testConfiguration) GetDefaultCache() configurationtypes.DefaultCacheInterface {
	return t.defaultCache
}
func (*testConfiguration) GetAPI() configurationtypes.API {
	return configurationtypes.API{}
}
func (*testConfiguration) GetLogLevel() string {
	return ""
}
func (*testConfiguration) GetLogger() *zap.Logger {
	return zap.NewNop()
}
func (*testConfiguration) SetLogger(*zap.Logger) {
}
func (*testConfiguration) GetYkeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}
func (*testConfiguration) GetSurrogateKeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}
func (t *testConfiguration) GetCacheKeys() configurationtypes.CacheKeys {
	return t.cacheKeys
}

func Test_CacheContext_SetupContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := testConfiguration{
		defaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	ctx := cacheContext{}

	ctx.SetupContext(&c)
	if ctx.cacheName != "Souin" {
		t.Error("The context must be equal to Souin.")
	}

	c.defaultCache.CacheName = "Something"
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
