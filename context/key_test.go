package context

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins/souin/configuration"
)

func Test_KeyContext_SetupContext(t *testing.T) {
	ctx := keyContext{}
	ctx.SetupContext(&configuration.Configuration{
		DefaultCache: &configurationtypes.DefaultCache{
			Key: configurationtypes.Key{},
		},
	})
	if ctx.disable_host {
		t.Errorf("The host must be disabled.")
	}
	if ctx.disable_method {
		t.Errorf("The method must be disabled.")
	}

	m := make(map[configurationtypes.RegValue]configurationtypes.Key)
	rg := configurationtypes.RegValue{
		Regexp: regexp.MustCompile(".*"),
	}
	m[rg] = configurationtypes.Key{
		DisableHost:   true,
		DisableMethod: true,
	}
	ctx.SetupContext(&configuration.Configuration{
		DefaultCache: &configurationtypes.DefaultCache{
			Key: configurationtypes.Key{
				DisableHost:   true,
				DisableMethod: true,
			},
		},
		CacheKeys: m,
	})

	if !ctx.disable_host {
		t.Errorf("The host must be enabled.")
	}
	if !ctx.disable_method {
		t.Errorf("The method must be enabled.")
	}
	if !ctx.overrides[rg.Regexp].disable_host {
		t.Errorf("The host must be enabled.")
	}
	if !ctx.overrides[rg.Regexp].disable_method {
		t.Errorf("The method must be enabled.")
	}
}

func Test_KeyContext_SetContext(t *testing.T) {
	ctx := keyContext{}
	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req = ctx.SetContext(req.WithContext(context.WithValue(req.Context(), HashBody, "-with_the_hash")))
	if req.Context().Value(Key).(string) != "GET-domain.com-/-with_the_hash" {
		t.Errorf("The Key context must be equal to GET-domain.com-/-with_the_hash, %s given.", req.Context().Value(Key).(string))
	}

	m := make(map[*regexp.Regexp]keyContext)
	m[regexp.MustCompile("/matched")] = keyContext{
		disable_host:   false,
		disable_method: true,
	}
	ctx2 := keyContext{
		disable_host:   true,
		disable_method: true,
		overrides:      m,
	}
	req2 := httptest.NewRequest(http.MethodGet, "http://domain.com/matched", nil)
	req2 = ctx2.SetContext(req2.WithContext(context.WithValue(req2.Context(), HashBody, "")))
	if req2.Context().Value(Key).(string) != "domain.com-/matched" {
		t.Errorf("The Key context must be equal to domain.com-/matched, %s given.", req2.Context().Value(Key).(string))
	}

	m = make(map[*regexp.Regexp]keyContext)
	m[regexp.MustCompile("/matched")] = keyContext{
		disable_host:   true,
		disable_method: false,
	}
	ctx3 := keyContext{
		disable_method: true,
		overrides:      m,
	}
	req3 := httptest.NewRequest(http.MethodGet, "http://domain.com/matched", nil)
	req3 = ctx3.SetContext(req3.WithContext(context.WithValue(req3.Context(), HashBody, "")))
	if req3.Context().Value(Key).(string) != "GET-/matched" {
		t.Errorf("The Key context must be equal to GET-/matched, %s given.", req3.Context().Value(Key).(string))
	}

	req4 := httptest.NewRequest(http.MethodGet, "http://domain.com/something", nil)
	req4 = ctx3.SetContext(req4.WithContext(context.WithValue(req4.Context(), HashBody, "")))
	if req4.Context().Value(Key).(string) != "domain.com-/something" {
		t.Errorf("The Key context must be equal to domain.com-/something, %s given.", req4.Context().Value(Key).(string))
	}
}
