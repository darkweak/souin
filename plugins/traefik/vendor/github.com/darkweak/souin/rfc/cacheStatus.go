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
			if !validateTimeHeader(headers, h, v) {
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
		setMalformedHeader(&h, "Cache-Control")
		r.Header = h

		return false
	}

	return true
}

// HitCache set hit and manage age header too
func HitCache(h *http.Header) {
	manageAge(h)
	if h.Get("Cache-Status") == "" {
		h.Set("Cache-Status", "Souin; fwd=hit; ttl="+h.Get("Age"))
	}
}

func manageAge(h *http.Header) {
	utc1 := time.Now().UTC()
	utc2 := utc1
	dh := h.Get("Date")
	if dh == "" {
		h.Set("Date", utc1.Format(http.TimeFormat))
	} else if validateTimeHeader(h, "Date", dh) {
		if u, e := http.ParseTime(h.Get("Date")); e == nil {
			utc2 = u
		} else {
			setMalformedHeader(h, "Date")
		}
	}

	h.Set("Age", ageToString(correctedInitialAge(utc1, utc2)))
}

func setMalformedHeader(headers *http.Header, header string) {
	SetRequestCacheStatus(headers, "MALFORMED-"+strings.ToUpper(header))
}

// SetCacheStatusEventually eventually set cache status header
func SetCacheStatusEventually(resp *http.Response) *http.Response {
	h := resp.Header
	validateEmptyHeaders(&h)
	manageAge(&h)

	resp.Header = h
	return resp
}
