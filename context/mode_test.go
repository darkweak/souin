package context

import (
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

func Test_ModeContext_SetupContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := testConfiguration{
		defaultCache: &dc,
	}
	c.SetLogger(zap.NewNop().Sugar())
	ctx := ModeContext{}

	ctx.SetupContext(&c)
	if ctx.Bypass_request || ctx.Bypass_response || !ctx.Strict {
		t.Error("The context must be strict and must not bypass either response or request.")
	}

	c.defaultCache.Mode = "bypass"
	ctx = ModeContext{}
	ctx.SetupContext(&c)
	if !ctx.Bypass_request || !ctx.Bypass_response || ctx.Strict {
		t.Error("The context must bypass either response and request.")
	}

	c.defaultCache.Mode = "bypass_request"
	ctx = ModeContext{}
	ctx.SetupContext(&c)
	if !ctx.Bypass_request || ctx.Bypass_response || ctx.Strict {
		t.Error("The context must bypass request only.")
	}

	c.defaultCache.Mode = "bypass_response"
	ctx = ModeContext{}
	ctx.SetupContext(&c)
	if ctx.Bypass_request || !ctx.Bypass_response || ctx.Strict {
		t.Error("The context must bypass response only.")
	}

	c.defaultCache.Mode = "strict"
	ctx = ModeContext{}
	ctx.SetupContext(&c)
	if ctx.Bypass_request || ctx.Bypass_response || !ctx.Strict {
		t.Error("The context must be strict.")
	}

	c.defaultCache.Mode = "default_value"
	ctx = ModeContext{}
	ctx.SetupContext(&c)
	if ctx.Bypass_request || ctx.Bypass_response || !ctx.Strict {
		t.Error("The context must be strict.")
	}
}
