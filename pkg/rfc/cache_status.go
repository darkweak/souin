package rfc

import (
	ctx "context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/darkweak/souin/context"
	"github.com/pquerna/cachecontrol/cacheobject"
)

const (
	StoredTTLHeader    = "X-Souin-Stored-TTL"
	StoredLengthHeader = "X-Souin-Stored-Length"
	// StoredExpiryHeader carries the instant a stored response stops being
	// fresh. A storer keeps a response longer than that so it can be
	// revalidated instead of refetched, so its own notion of freshness is
	// deliberately looser and this header is what decides.
	StoredExpiryHeader = "X-Souin-Stored-Expiry"
)

// SetStoredExpiry records when the stored response stops being fresh.
func SetStoredExpiry(h http.Header, expiry time.Time) {
	h.Set(StoredExpiryHeader, expiry.Format(time.RFC3339Nano))
}

// StoredExpiry returns the instant the stored response stops being fresh, and
// whether it could be read at all.
func StoredExpiry(h http.Header) (time.Time, bool) {
	expiry, err := time.Parse(time.RFC3339Nano, h.Get(StoredExpiryHeader))

	return expiry, err == nil
}

// IsStoredFresh tells whether a stored response may still be reused without
// being revalidated first.
func IsStoredFresh(h http.Header, now time.Time) bool {
	expiry, ok := StoredExpiry(h)

	return !ok || now.Before(expiry)
}

// StoredStaleness returns for how long a stored response has been stale.
func StoredStaleness(h http.Header, now time.Time) time.Duration {
	expiry, ok := StoredExpiry(h)
	if !ok {
		return 0
	}

	return now.Sub(expiry)
}

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
func ValidateCacheControl(r *http.Response, requestCc *cacheobject.RequestCacheDirectives) bool {
	if _, err := cacheobject.ParseResponseCacheControl(HeaderAllCommaSepValuesString(r.Header, "Cache-Control")); err != nil {
		h := r.Header
		setMalformedHeader(&h, "CACHE-CONTROL", r.Request.Context().Value(context.CacheName).(string))
		r.Header = h

		return false
	}

	if requestCc.MinFresh >= 0 {
		t, e := http.ParseTime(r.Header.Get("Date"))
		return e == nil && int(time.Since(t).Seconds()) > int(requestCc.MinFresh)
	}

	return true
}

func GetCacheKeyFromCtx(currentCtx ctx.Context) string {
	if displayable := currentCtx.Value(context.DisplayableKey); displayable != nil && displayable.(bool) {
		if key := currentCtx.Value(context.Key); key != nil {
			return key.(string)
		}
	}

	return "*****"
}

// MissCache set miss fwd
// func MissCache(set func(key, value string), req *http.Request, reason string) {
// 	set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=%s", req.Context().Value(context.CacheName), GetCacheKeyFromCtx(req.Context()), reason))
// }

// HitStaleCache set hit and stale in the Cache-Status header
func HitStaleCache(h *http.Header) {
	h.Set("Cache-Status", h.Get("Cache-Status")+"; fwd=stale")
}

func manageAge(h *http.Header, ttl time.Duration, cacheName, key, storerName string) {
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

	if h.Get(StoredTTLHeader) != "" {
		ttl, _ = time.ParseDuration(h.Get(StoredTTLHeader))
		h.Del(StoredTTLHeader)
	}
	h.Del(StoredExpiryHeader)

	apparentAge := utc1.Sub(utc2)
	if apparentAge < 0 {
		apparentAge = 0
	}

	var oldAge int
	{
		var err error
		oldAgeString := h.Get("Age")
		oldAge, err = strconv.Atoi(oldAgeString)
		if err != nil {
			oldAge = 0
		}
	}

	cage := int(math.Ceil(apparentAge.Seconds()))
	age := strconv.Itoa(oldAge + cage)
	h.Set("Age", age)
	ttlValue := strconv.Itoa(int(ttl.Seconds()) - cage)
	h.Set("Cache-Status", cacheName+"; hit; ttl="+ttlValue+"; key="+key+"; detail="+storerName)
}

func setMalformedHeader(headers *http.Header, header, cacheName string) {
	SetRequestCacheStatus(headers, "MALFORMED-"+header, cacheName)
}

// SetCacheStatusHeader set the Cache-Status header
func SetCacheStatusHeader(resp *http.Response, storerName string) *http.Response {
	h := resp.Header
	cacheName := resp.Request.Context().Value(context.CacheName).(string)
	validateEmptyHeaders(&h, cacheName)
	manageAge(&h, 0, cacheName, GetCacheKeyFromCtx(resp.Request.Context()), storerName)

	resp.Header = h
	return resp
}
