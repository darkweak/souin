package rfc

import (
	"github.com/pquerna/cachecontrol/cacheobject"
	"net/http"
	"strings"
	"time"
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
	h.Set("Cache-Status", "Souin; fwd=hit")
	manageAge(h)
}

func manageAge(h *http.Header) {
	var rt time.Time

	utc1 := time.Now().UTC()
	utc2 := utc1
	date := h.Get("Date")
	if date == "" {
		h.Set("Date", utc1.Format(http.TimeFormat))
	} else if validateTimeHeader(h, "Date", date) {
		if u, e := http.ParseTime(h.Get("Date")); e == nil {
			utc2 = u
		} else {
			setMalformedHeader(h, "Date")
		}
	}

	ageValue := h.Get("Age")
	correctedInitialAge := correctedInitialAge(utc1, utc2, rt, ageValue)

	if ageValue != "" || correctedInitialAge != 0 {
		h.Set("Age", ageToString(correctedInitialAge))
	}
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
