package rfc

import (
	"fmt"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"net/http"
	"net/http/httptest"
	"testing"
)

func commonInitializer() (*http.Request, types.AbstractProviderInterface) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	httptest.NewRecorder()

	return r, prs
}

func TestCachedResponse_WithUpdate(t *testing.T) {
	r, c := commonInitializer()
	res, err := CachedResponse(c, r, GetCacheKey(r), NewTransport(c), true)

	if err != nil {
		errors.GenerateError(t, "CachedResponse cannot throw error")
	}

	if res.Response != nil {
		errors.GenerateError(t, fmt.Sprintf("Result from cached response should be a valid response"))
	}

	if res.Request != nil || res.Proxy != nil {
		errors.GenerateError(t, fmt.Sprintf("Request and Proxy shouldn't be set"))
	}
}

func TestCachedResponse_WithoutUpdate(t *testing.T) {
	r, c := commonInitializer()
	key := GetCacheKey(r)
	res, err := CachedResponse(c, r, key, NewTransport(c), false)

	if err != nil {
		errors.GenerateError(t, "CachedResponse cannot throw error")
	}

	if res.Response != nil {
		errors.GenerateError(t, fmt.Sprintf("Result from cached response should be a valid response"))
	}

	if res.Request != nil || res.Proxy != nil {
		errors.GenerateError(t, fmt.Sprintf("Request and Proxy shouldn't be set"))
	}
}
