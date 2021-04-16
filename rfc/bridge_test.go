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

func commonInitializer() (*http.Request, map[string]types.AbstractProviderInterface) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	httptest.NewRecorder()

	return r, prs
}

func TestCachedResponse_WithUpdate(t *testing.T) {
	r, c := commonInitializer()
	defer c["olric"].Reset()

	for _, v := range c {
		res, err := CachedResponse(v, r, GetCacheKey(r), NewTransport(c), true)

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
}

func TestCachedResponse_WithoutUpdate(t *testing.T) {
	r, c := commonInitializer()
	defer c["olric"].Reset()

	for _, v := range c {
		res, err := CachedResponse(v, r, GetCacheKey(r), NewTransport(c), false)

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
}
