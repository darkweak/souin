package rfc

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/darkweak/souin/errors"
)

func TestHitCache(t *testing.T) {
	h := http.Header{}

	h.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	h.Set("Age", "1")

	HitCache(&h, 4*time.Second)
	if h.Get("Cache-Status") == "" || h.Get("Cache-Status") != "Souin; hit; ttl=3" {
		errors.GenerateError(t, fmt.Sprintf("Cache-Status cannot be null when hit and must match hit, %s given", h.Get("Cache-Status")))
	}
	if ti, e := http.ParseTime(h.Get("Date")); h.Get("Date") == "" || e != nil || h.Get("Date") != ti.Format(http.TimeFormat) {
		errors.GenerateError(t, fmt.Sprintf("Date cannot be null when invalid and must match %s, %s given", h.Get("Date"), ti.Format(http.TimeFormat)))
	}

	h.Set("Date", "Invalid")
	HitCache(&h, 0)
	if h.Get("Cache-Status") == "" || h.Get("Cache-Status") != "Souin; fwd=request; detail=MALFORMED-DATE" {
		errors.GenerateError(t, fmt.Sprintf("Cache-Status cannot be null when hit and must match MALFORMED-DATE, %s given", h.Get("Cache-Status")))
	}
	if h.Get("Date") == "" || h.Get("Date") != "Invalid" {
		errors.GenerateError(t, fmt.Sprintf("Date cannot be null when invalid and must match Invalid, %s given", h.Get("Date")))
	}
}

func TestSetRequestCacheStatus(t *testing.T) {
	h := http.Header{}

	SetRequestCacheStatus(&h, "AHeader")
	if h.Get("Cache-Status") != "Souin; fwd=request; detail=AHeader" {
		errors.GenerateError(t, fmt.Sprintf("The Cache-Status must match %s, %s given", "Souin; fwd=request; detail=AHeader", h.Get("Cache-Status")))
	}
	SetRequestCacheStatus(&h, "")
	if h.Get("Cache-Status") != "Souin; fwd=request; detail=" {
		errors.GenerateError(t, fmt.Sprintf("The Cache-Status must match %s, %s given", "Souin; fwd=request; detail=", h.Get("Cache-Status")))
	}
	SetRequestCacheStatus(&h, "A very long header with spaces")
	if h.Get("Cache-Status") != "Souin; fwd=request; detail=A very long header with spaces" {
		errors.GenerateError(t, fmt.Sprintf("The Cache-Status must match %s, %s given", "Souin; fwd=request; detail=A very long header with spaces", h.Get("Cache-Status")))
	}
}

func TestValidateCacheControl(t *testing.T) {
	r := http.Response{}
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

func TestSetCacheStatusEventually(t *testing.T) {
	r := http.Response{}
	r.Header = http.Header{}

	SetCacheStatusEventually(&r)
	if r.Header.Get("Cache-Status") != "Souin; hit; ttl=-1" {
		errors.GenerateError(t, fmt.Sprintf("The Cache-Status should be equal to Souin; hit; ttl=-1, %s given", r.Header.Get("Cache-Status")))
	}

	r.Header = http.Header{"Date": []string{"Invalid"}}
	SetCacheStatusEventually(&r)
	if r.Header.Get("Cache-Status") == "" {
		errors.GenerateError(t, "The Cache-Control shouldn't be empty")
	}
	if r.Header.Get("Cache-Status") != "Souin; fwd=request; detail=MALFORMED-DATE" {
		errors.GenerateError(t, "The Cache-Control should be equal to MALFORMED-DATE")
	}

	r.Header = http.Header{}
	SetCacheStatusEventually(&r)
	if ti, e := http.ParseTime(r.Header.Get("Date")); r.Header.Get("Date") == "" || e != nil || r.Header.Get("Date") != ti.Format(http.TimeFormat) {
		errors.GenerateError(t, fmt.Sprintf("Date cannot be null when invalid and must match %s, %s given", r.Header.Get("Date"), ti.Format(http.TimeFormat)))
	}
}
