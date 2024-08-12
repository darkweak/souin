package context

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

func Test_MethodContext_SetupContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := testConfiguration{
		defaultCache: &dc,
	}
	c.SetLogger(zap.NewNop().Sugar())
	ctx := methodContext{}

	ctx.SetupContext(&c)
	if ctx.custom {
		t.Error("The context must not be custom if no allowed HTTP verbs are set in the configuration.")
	}

	c.defaultCache.AllowedHTTPVerbs = []string{http.MethodGet}
	ctx.SetupContext(&c)
	if !ctx.custom {
		t.Error("The context must be custom if at least one allowed HTTP verb is set in the configuration.")
	}
}

func Test_MethodContext_SetContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := testConfiguration{
		defaultCache: &dc,
	}
	c.SetLogger(zap.NewNop().Sugar())
	ctx := methodContext{}
	c.defaultCache.AllowedHTTPVerbs = []string{http.MethodGet, http.MethodHead}
	ctx.SetupContext(&c)

	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req = ctx.SetContext(req)
	if !req.Context().Value(SupportedMethod).(bool) {
		t.Error("The SupportedMethod context must be true.")
	}

	req = httptest.NewRequest(http.MethodDelete, "http://domain.com", nil)
	req = ctx.SetContext(req)
	if req.Context().Value(SupportedMethod).(bool) {
		t.Error("The SupportedMethod context must be false.")
	}
}
