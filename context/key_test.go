package context

import (
	"context"
	"net/http"
	"net/http/httptest"
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
}

func Test_KeyContext_SetContext(t *testing.T) {
	ctx := keyContext{}
	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req = ctx.SetContext(req.WithContext(context.WithValue(req.Context(), HashBody, "-with_the_hash")))
	if req.Context().Value(Key).(string) != "GET-domain.com-http://domain.com-with_the_hash" {
		t.Errorf("The Key context must be equal to GET-domain.com-http://domain.com-with_the_hash, %s given.", req.Context().Value(Key).(string))
	}
}
