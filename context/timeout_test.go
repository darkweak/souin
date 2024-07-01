package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

func Test_TimeoutContext_SetupContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := testConfiguration{
		defaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	ctx := timeoutContext{}

	ctx.SetupContext(&c)
	if ctx.timeoutBackend != defaultTimeoutBackend {
		t.Error("The timeout backend must be equal to the default timeout backend when no directives are given.")
	}
	if ctx.timeoutCache != defaultTimeoutCache {
		t.Error("The timeout cache must be equal to the default timeout cache when no directives are given.")
	}

	c.defaultCache.Timeout.Backend = configurationtypes.Duration{Duration: time.Second}
	ctx.SetupContext(&c)
	if ctx.timeoutBackend != time.Second {
		t.Error("The timeout backend must be equal to one second when a directive is given.")
	}
	if ctx.timeoutCache != defaultTimeoutCache {
		t.Error("The timeout cache must be equal to the default timeout cache when no directives are given.")
	}

	c.defaultCache.Timeout = configurationtypes.Timeout{}
	c.defaultCache.Timeout.Cache = configurationtypes.Duration{Duration: time.Second}
	ctx.SetupContext(&c)
	if ctx.timeoutBackend != defaultTimeoutBackend {
		t.Error("The timeout backend must be equal to 10 seconds when no directives are given.")
	}
	if ctx.timeoutCache != time.Second {
		t.Error("The timeout cache must be equal to one second when a directive is given.")
	}
}

func Test_TimeoutContext_SetContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{
		Timeout: configurationtypes.Timeout{
			Backend: configurationtypes.Duration{
				Duration: time.Second,
			},
			Cache: configurationtypes.Duration{
				Duration: time.Millisecond,
			},
		},
	}
	c := testConfiguration{
		defaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	ctx := timeoutContext{}
	ctx.SetupContext(&c)

	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req = ctx.SetContext(req)
	if req.Context().Value(TimeoutCache).(time.Duration) != time.Millisecond {
		t.Error("The TimeoutCache context must be set in the request.")
	}
	if req.Context().Value(TimeoutCancel).(context.CancelFunc) == nil {
		t.Error("The TimeoutCancel context must be set and be a CancelFunc.")
	}
}
