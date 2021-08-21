package rfc

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

func TestIsVaryCacheable(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "http://domain.com/testing", nil)
	if IsVaryCacheable(r) {
		errors.GenerateError(t, "It should return true")
	}

	r.Method = http.MethodHead
	if !IsVaryCacheable(r) {
		errors.GenerateError(t, "It should return true")
	}

	r.Method = http.MethodGet
	if !IsVaryCacheable(r) {
		errors.GenerateError(t, "It should return true")
	}

	r.Header.Set("range", "bad")
	if IsVaryCacheable(r) {
		errors.GenerateError(t, "It should return false")
	}

	r.Method = http.MethodHead
	if IsVaryCacheable(r) {
		errors.GenerateError(t, "It should return false")
	}
}

func TestVaryTransport_GetProvider(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)

	tr := NewTransport(prs, ykeys.InitializeYKeys(c.Ykeys))
	if tr.GetProvider() == nil {
		errors.GenerateError(t, "Provider should exist")
	}
}

func TestVaryTransport_SetURL(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(config)
	matchedURL := configurationtypes.URL{
		TTL:     configurationtypes.Duration{Duration: config.GetDefaultCache().GetTTL()},
		Headers: config.GetDefaultCache().GetHeaders(),
	}

	tr := NewTransport(prs, ykeys.InitializeYKeys(config.Ykeys))
	tr.SetURL(matchedURL)

	if len(tr.ConfigurationURL.Headers) != len(matchedURL.Headers) || tr.ConfigurationURL.TTL != matchedURL.TTL {
		errors.GenerateError(t, "The transport configurationURL property must be a shallow copy of the matchedURL")
	}
}

func TestVaryTransport_SetCache(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://domain.com/testing", nil)
	res := httptest.NewRecorder().Result()
	key := GetCacheKey(req)
	config := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(config)
	tr := NewTransport(prs, ykeys.InitializeYKeys(config.Ykeys))
	tr.SetCache(key, res)
	time.Sleep(1 * time.Second)
	if v, e := tr.YkeyStorage.Get("The_Third_Test"); v.(string) != key || !e {
		errors.GenerateError(t, fmt.Sprintf("The url %s should be part of the %s tag", key, "The_Third_Test"))
	}
	if v, e := tr.YkeyStorage.Get("The_First_Test"); v.(string) != "" || !e {
		errors.GenerateError(t, fmt.Sprintf("The url %s shouldn't be part of the %s tag", key, "The_First_Test"))
	}
	if v, e := tr.YkeyStorage.Get("The_Second_Test"); v.(string) != "" || !e {
		errors.GenerateError(t, fmt.Sprintf("The url %s shouldn't be part of the %s tag", key, "The_Second_Test"))
	}
}
