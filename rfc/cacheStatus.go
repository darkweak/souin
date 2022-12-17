package rfc

import (
	ctx "context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/darkweak/souin/context"
	"github.com/pquerna/cachecontrol/cacheobject"
)

const storedTTLHeader = "X-Souin-Stored-TTL"

var emptyHeaders = []string{"Expires", "Last-Modified"}

func validateTimeHeader(headers *http.Header, h, t, cacheName string) bool {
	if _, err := http.ParseTime(t); err != nil {
		setMalformedHeader(headers, h, cacheName)
		return false
	}
	return true
}

func validateEmptyHeaders(headers *http.Header, cacheName string) {
	for _, h := range emptyHeaders {
		if v := headers.Get(h); v != "" {
			if !validateTimeHeader(headers, strings.ToUpper(h), v, cacheName) {
				return
			}
		}
	}
}

// SetRequestCacheStatus set the Cache-Status fwd=request
func SetRequestCacheStatus(h *http.Header, header, cacheName string) {
	h.Set("Cache-Status", cacheName+"; fwd=request; detail="+header)
}

// ValidateCacheControl check the Cache-Control header
func ValidateCacheControl(r *http.Response) bool {
	if _, err := cacheobject.ParseResponseCacheControl(r.Header.Get("Cache-Control")); err != nil {
		h := r.Header
		setMalformedHeader(&h, "CACHE-CONTROL", r.Request.Context().Value(context.CacheName).(string))
		r.Header = h

		return false
	}

	return true
}

func getCacheKeyFromCtx(currentCtx ctx.Context) string {
	key := currentCtx.Value(context.Key)
	displayable := currentCtx.Value(context.DisplayableKey)
	if key == nil || displayable == nil || !displayable.(bool) {
		return ""
	}

	return key.(string)
}

// MissCache set miss fwd
func MissCache(set func(key, value string), req *http.Request, reason string) {
	set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=%s", req.Context().Value(context.CacheName), getCacheKeyFromCtx(req.Context()), reason))
}

// HitStaleCache set hit and manage age header too
func HitStaleCache(h *http.Header, ttl time.Duration) {
	h.Set("Cache-Status", h.Get("Cache-Status")+"; fwd=stale")
}

func manageAge(h *http.Header, ttl time.Duration, cacheName, key string) {
	utc1 := time.Now().UTC()
	dh := h.Get("Date")
	if dh == "" {
		h.Set("Date", utc1.Format(http.TimeFormat))
	} else if !validateTimeHeader(h, "DATE", dh, cacheName) {
		return
	}

	var utc2 time.Time
	var e error
	if utc2, e = http.ParseTime(h.Get("Date")); e != nil {
		return
	}

	if h.Get(storedTTLHeader) != "" {
		ttl, _ = time.ParseDuration(h.Get(storedTTLHeader))
		h.Del(storedTTLHeader)
	}

	cage := correctedInitialAge(utc1, utc2)
	age := strconv.Itoa(cage)
	h.Set("Age", age)
	ttlValue := strconv.Itoa(int(ttl.Seconds()) - cage)
	h.Set("Cache-Status", cacheName+"; hit; ttl="+ttlValue+"; key="+key)
}

func setMalformedHeader(headers *http.Header, header, cacheName string) {
	SetRequestCacheStatus(headers, "MALFORMED-"+header, cacheName)
}

// SetCacheStatusEventually eventually set cache status header
func SetCacheStatusEventually(resp *http.Response) *http.Response {
	h := resp.Header
	cacheName := resp.Request.Context().Value(context.CacheName).(string)
	validateEmptyHeaders(&h, cacheName)
	manageAge(&h, 0, cacheName, getCacheKeyFromCtx(resp.Request.Context()))

	resp.Header = h
	return resp
}
