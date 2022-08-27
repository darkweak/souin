package rfc

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/context"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

func TestVaryMatches(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	tr := NewTransport(prs, ykeys.InitializeYKeys(c.Ykeys), surrogate.InitializeSurrogate(c))

	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder().Result()
	co := context.GetContext()
	co.Init(c)
	r = co.SetContext(r)

	if !varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return true if no header sent")
	}
	if !validateVary(r, res, r.Context().Value(context.Key).(string), tr) {
		errors.GenerateError(t, fmt.Sprintf("It doesn't contain vary header in the Response. It should validate it, %v given", res.Header))
	}

	header := "Cache"
	r.Header.Set(header, "same")
	res.Header.Set("vary", header)

	if !varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return true if Response contains a vary header that is not null in the request")
	}

	if !validateVary(r, res, r.Context().Value(context.Key).(string), tr) {
		errors.GenerateError(t, fmt.Sprintf("It contains valid vary headers in the Response. It should validate it, %v given", res.Header))
	}

	r.Header.Set(header, "")

	if varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return false if Response contains a vary header that is empty in the request")
	}

	if !validateVary(r, res, r.Context().Value(context.Key).(string), tr) {
		errors.GenerateError(t, fmt.Sprintf("It contains valid vary headers in the Response. It should validate it, %v given", res.Header))
	}

	if len(prs.Get(r.Context().Value(context.Key).(string))) == 0 {
		errors.GenerateError(t, fmt.Sprintf("The key %s should exist in the storage provider. %v given", r.Context().Value(context.Key).(string), prs.Get(r.Context().Value(context.Key).(string))))
	}

	variedHeaders := headerAllCommaSepValues(res.Header)
	variedCacheKey := GetVariedCacheKey(r, variedHeaders)
	b := prs.Get(GetVariedCacheKey(r, headerAllCommaSepValues(res.Header)))
	if len(b) == 0 {
		errors.GenerateError(t, fmt.Sprintf("The key %s with headers %v should exist in the storage provider. %v given", variedCacheKey, variedHeaders, b))
	}
}
