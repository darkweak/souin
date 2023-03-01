package rfc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	souinCtx "github.com/darkweak/souin/context"
	"github.com/darkweak/souin/errors"
)

func TestSetRequestCacheStatus(t *testing.T) {
	h := http.Header{}

	SetRequestCacheStatus(&h, "AHeader", "Souin")
	if h.Get("Cache-Status") != "Souin; fwd=request; detail=AHeader" {
		errors.GenerateError(t, fmt.Sprintf("The Cache-Status must match %s, %s given", "Souin; fwd=request; detail=AHeader", h.Get("Cache-Status")))
	}
	SetRequestCacheStatus(&h, "", "Souin")
	if h.Get("Cache-Status") != "Souin; fwd=request; detail=" {
		errors.GenerateError(t, fmt.Sprintf("The Cache-Status must match %s, %s given", "Souin; fwd=request; detail=", h.Get("Cache-Status")))
	}
	SetRequestCacheStatus(&h, "A very long header with spaces", "Souin")
	if h.Get("Cache-Status") != "Souin; fwd=request; detail=A very long header with spaces" {
		errors.GenerateError(t, fmt.Sprintf("The Cache-Status must match %s, %s given", "Souin; fwd=request; detail=A very long header with spaces", h.Get("Cache-Status")))
	}
}

func TestValidateCacheControl(t *testing.T) {
	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rq = rq.WithContext(context.WithValue(rq.Context(), souinCtx.CacheName, "Souin"))
	r := http.Response{
		Request: rq,
	}
	r.Header = http.Header{}

	valid := ValidateCacheControl(&r)
	if !valid {
		errors.GenerateError(t, "The Cache-Control should be valid while an empty string is provided")
	}
	h := http.Header{
		"Cache-Control": []string{"stale-if-error;malformed"},
	}
	r.Header = h
	valid = ValidateCacheControl(&r)
	if valid {
		errors.GenerateError(t, "The Cache-Control shouldn't be valid with max-age")
	}
}

func TestGetCacheKeyFromCtx(t *testing.T) {
	if GetCacheKeyFromCtx(context.WithValue(context.WithValue(context.Background(), souinCtx.Key, "MyKey"), souinCtx.DisplayableKey, true)) != "MyKey" {
		t.Error("GetCacheKeyFromCtx must return the key when displayable")
	}
	if GetCacheKeyFromCtx(context.WithValue(context.WithValue(context.Background(), souinCtx.Key, "MyKey"), souinCtx.DisplayableKey, false)) != "" {
		t.Error("GetCacheKeyFromCtx must not return the key when hidden")
	}
}

func TestHitStaleCache(t *testing.T) {
	h := http.Header{
		"Cache-Status": []string{"previous value"},
	}
	HitStaleCache(&h)
	if h.Get("Cache-Status") != "previous value; fwd=stale" {
		t.Error("HitStaleCache must append the stale directive in the Cache-Status HTTP header")
	}
}
