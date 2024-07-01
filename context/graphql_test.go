package context

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

func Test_GraphQLContext_SetupContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := testConfiguration{
		defaultCache: &dc,
	}
	ctx := graphQLContext{}

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

func Test_GraphQLContext_SetContext(t *testing.T) {
	dc := configurationtypes.DefaultCache{}
	c := testConfiguration{
		defaultCache: &dc,
	}
	c.SetLogger(zap.NewNop())
	ctx := graphQLContext{custom: true}

	req := httptest.NewRequest(http.MethodGet, "http://domain.com", nil)
	req.Body = nil
	req = ctx.SetContext(req)
	if req.Context().Value(HashBody).(string) != "" {
		t.Error("The HashBody must not be set in the context request.")
	}

	req = httptest.NewRequest(http.MethodGet, "http://domain.com", bytes.NewBuffer([]byte("{something}")))
	req = ctx.SetContext(req)
	if req.Context().Value(HashBody).(string) != "-d3f2a4350803c933ff32c6b14a353df36580bed4e0b45712c667266f8e219300" {
		t.Error("The HashBody must be set in the context request.")
	}

	req = httptest.NewRequest(http.MethodGet, "http://domain.com", bytes.NewBuffer([]byte(`{"query":"mutation":}`)))
	req = ctx.SetContext(req)
	if req.Context().Value(HashBody).(string) != "" {
		t.Error("The HashBody must not be set in the context request if mutation.")
	}
	if !req.Context().Value(IsMutationRequest).(bool) {
		t.Error("The request must be detected as a mutation.")
	}
}
