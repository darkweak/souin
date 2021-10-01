package rfc

import (
	"net/http"
	"strings"
	"time"

	"github.com/pquerna/cachecontrol/cacheobject"
)

var emptyHeaders = []string{"Expires", "Last-Modified"}

func validateTimeHeader(headers *http.Header, h string, t string) bool {
	if _, err := http.ParseTime(t); err != nil {
		setMalformedHeader(headers, h)
		return false
	}
	return true
}

func validateEmptyHeaders(headers *http.Header) {
	for _, h := range emptyHeaders {
		if v := headers.Get(h); v != "" {
			if !validateTimeHeader(headers, strings.ToUpper(h), v) {
				return
			}
		}
	}
}

// SetRequestCacheStatus set the Cache-Status fwd=request
func SetRequestCacheStatus(h *http.Header, header string) {
	h.Set("Cache-Status", "Souin; fwd=request; detail="+header)
}

// ValidateCacheControl check the Cache-Control header
func ValidateCacheControl(r *http.Response) bool {
	if _, err := cacheobject.ParseResponseCacheControl(r.Header.Get("Cache-Control")); err != nil {
		h := r.Header
		setMalformedHeader(&h, "CACHE-CONTROL")
		r.Header = h

		return false
	}

	return true
}

// HitCache set hit and manage age header too
func HitCache(h *http.Header) {
	manageAge(h)
}

// HitStaleCache set hit and manage age header too
func HitStaleCache(h *http.Header) {
	manageAge(h)
	h.Set("Cache-Status", h.Get("Cache-Status")+"; fwd=stale")
}

func manageAge(h *http.Header) {
	utc1 := time.Now().UTC()
	dh := h.Get("Date")
	if dh == "" {
		h.Set("Date", utc1.Format(http.TimeFormat))
	} else if !validateTimeHeader(h, "DATE", dh) {
		return
	}

	var utc2 time.Time
	var e error
	if utc2, e = http.ParseTime(h.Get("Date")); e != nil {
		return
	}

	age := ageToString(correctedInitialAge(utc1, utc2))
	h.Set("Age", age)
	h.Set("Cache-Status", "Souin; fwd=hit; ttl="+h.Get("Age"))
}

func setMalformedHeader(headers *http.Header, header string) {
	SetRequestCacheStatus(headers, "MALFORMED-"+header)
}

// SetCacheStatusEventually eventually set cache status header
func SetCacheStatusEventually(resp *http.Response) *http.Response {
	h := resp.Header
	validateEmptyHeaders(&h)
	manageAge(&h)

	resp.Header = h
	return resp
}
