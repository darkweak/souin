package rfc

import (
	"fmt"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"net/http/httptest"
	"testing"
)

func TestVaryMatches(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	tr := NewTransport(prs)

	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder().Result()

	if !varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return true if no header sent")
	}
	if !validateVary(r, res, GetCacheKey(r), tr) {
		errors.GenerateError(t, fmt.Sprintf("It doesn't contain vary header in the Response. It should validate it, %v given", res.Header))
	}

	header := "Cache"
	r.Header.Set(header, "same")
	res.Header.Set("vary", header)

	if !varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return true if Response contains a vary header that is not null in the request")
	}

	if !validateVary(r, res, GetCacheKey(r), tr) {
		errors.GenerateError(t, fmt.Sprintf("It contains valid vary headers in the Response. It should validate it, %v given", res.Header))
	}

	r.Header.Set(header, "")

	if varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return false if Response contains a vary header that is empty in the request")
	}

	if !validateVary(r, res, GetCacheKey(r), tr) {
		errors.GenerateError(t, fmt.Sprintf("It contains valid vary headers in the Response. It should validate it, %v given", res.Header))
	}

	for n, p := range prs {
		if len(p.Get(GetCacheKey(r))) != 0 {
			errors.GenerateError(t, fmt.Sprintf("The key %s shouldn't exist in the %s provider. %v given", GetCacheKey(r), n, p.Get(GetCacheKey(r))))
		}
	}

	variedHeaders := headerAllCommaSepValues(res.Header, "vary")
	variedCacheKey := GetVariedCacheKey(r, variedHeaders)
	for n, p := range prs {
		b := p.Get(GetVariedCacheKey(r, headerAllCommaSepValues(res.Header, "vary")))
		if len(b) != 0 {
			errors.GenerateError(t, fmt.Sprintf("The key %s with headers %v shouldn't exist in the %s provider. %v given", variedCacheKey, variedHeaders, n, b))
		}
	}
}
