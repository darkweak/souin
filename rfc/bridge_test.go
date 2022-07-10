package rfc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/context"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

func commonInitializer() (*http.Request, types.AbstractProviderInterface, *VaryTransport) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	co := context.GetContext()
	co.Init(c)
	r = co.SetContext(r)
	httptest.NewRecorder()

	return r, prs, NewTransport(prs, ykeys.InitializeYKeys(c.Ykeys), surrogate.InitializeSurrogate(c))
}

func TestCachedResponse_WithUpdate(t *testing.T) {
	r, c, v := commonInitializer()

	res, _, err := CachedResponse(c, r, r.Context().Value(context.Key).(string), v)

	if err != nil {
		errors.GenerateError(t, "CachedResponse cannot throw error")
	}

	if res != nil {
		errors.GenerateError(t, "Result from cached response should be a valid response")
	}
}

func TestCachedResponse_WithoutUpdate(t *testing.T) {
	r, c, v := commonInitializer()
	res, _, err := CachedResponse(c, r, r.Context().Value(context.Key).(string), v)

	if err != nil {
		errors.GenerateError(t, "CachedResponse cannot throw error")
	}

	if res != nil {
		errors.GenerateError(t, "Result from cached response should be a valid response")
	}
}
