package rfc

import (
	"bufio"
	"bytes"
	ctx "context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/context"
)

const (
	stale = iota
	fresh
	transparent
)

// CachedResponse returns the cached http.Response for req if present, and nil
// otherwise.
func CachedResponse(c types.AbstractProviderInterface, req *http.Request, cachedKey string, transport types.TransportInterface) (*http.Response, bool, error) {
	clonedReq := cloneRequest(req)
	cachedVal := c.Prefix(cachedKey, req)
	response, _ := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(cachedVal)), clonedReq)

	if response != nil && ValidateCacheControl(response) {
		SetCacheStatusEventually(response)
		return ValidateMaxAgeCachedResponse(req, response), false, nil
	} else if response == nil {
		staleCachedVal := c.Prefix("STALE_"+cachedKey, req)
		response, _ = http.ReadResponse(bufio.NewReader(bytes.NewBuffer(staleCachedVal)), clonedReq)
		if nil != response && ValidateCacheControl(response) {
			addTime, _ := time.ParseDuration(response.Header.Get(storedTTLHeader))
			SetCacheStatusEventually(response)
			return ValidateMaxAgeCachedStaleResponse(req, response, int(addTime.Seconds())), true, nil
		}
	}
	return nil, false, nil
}

func commonCacheControl(req *http.Request, t http.RoundTripper, withRoundTrip bool) (*http.Response, error) {
	reqCacheControl := parseCacheControl(req.Header)
	if _, ok := reqCacheControl["only-if-cached"]; ok {
		return newGatewayTimeoutResponse(req), nil
	}

	if !withRoundTrip {
		return req.Response, nil
	}

	return t.RoundTrip(req)
}

func (t *VaryTransport) deleteCache(key string) {
	go func() {
		if t.Transport.YkeyStorage != nil {
			t.Transport.YkeyStorage.InvalidateTagURLs(key)
		}
	}()
	t.Transport.Provider.Delete(key)
}

func canBypassHeader(headers http.Header, bypassed []string) bool {
	if headers.Get("Authorization") == "" {
		return true
	}

	for _, header := range bypassed {
		if strings.ToLower(header) == "authorization" {
			return true
		}
	}

	return false
}

// BaseRoundTrip is the base for RoundTrip
func (t *VaryTransport) BaseRoundTrip(req *http.Request) (string, bool, *http.Response) {
	cacheKey := req.Context().Value(context.Key).(string)
	_, err := req.Cookie("authorization")
	fmt.Println("IGNORED HEADERS =>", req.Context().Value(context.IgnoredHeaders).([]string))
	cacheable := IsVaryCacheable(req) && canBypassHeader(req.Header, req.Context().Value(context.IgnoredHeaders).([]string)) && err != nil

	if !cacheable {
		go func() {
			t.CoalescingLayerStorage.Set(cacheKey)
		}()
		t.deleteCache(cacheKey)
	}

	return cacheKey, cacheable, req.Response
}

func commonVaryMatchesVerification(cachedResp *http.Response, req *http.Request) *http.Response {
	if varyMatches(cachedResp, req) {
		// Can only use cached value if the new request doesn't Vary significantly
		freshness := getFreshness(cachedResp.Header, req.Header)
		if freshness == fresh {
			return cachedResp
		}

		if freshness == stale {
			var req2 *http.Request
			// Add validators if caller hasn't already done so
			etag := cachedResp.Header.Get("etag")
			if etag != "" && req.Header.Get("etag") == "" {
				req2 = cloneRequest(req)
				req2.Header.Set("if-none-match", etag)
			}
			lastModified := cachedResp.Header.Get("last-modified")
			if lastModified != "" && req.Header.Get("last-modified") == "" {
				if req2 == nil {
					req2 = cloneRequest(req)
				}
				req2.Header.Set("if-modified-since", lastModified)
			}
			if req2 != nil {
				req = req2 // nolint
			}
		}
	}

	return nil
}

func getAppropriateCacheControlHeader(h http.Header, cacheName string) string {
	if v := h.Get(cacheName + "-Cache-Control"); v != "" {
		return v
	}
	if v := h.Get("CDN-Cache-Control"); v != "" {
		return v
	}
	return h.Get("Cache-Control")
}

// UpdateCacheEventually will handle Request and update the previous one in the cache provider
func (t *VaryTransport) UpdateCacheEventually(req *http.Request) (*http.Response, error) {
	select {
	case <-req.Context().Done():
		return nil, nil
	default:
	}
	ccHeaderValue := getAppropriateCacheControlHeader(req.Response.Header, req.Context().Value(context.CacheName).(string))
	req = req.WithContext(ctx.WithValue(req.Context(), context.CacheControlCtx, ccHeaderValue))
	if ccHeaderValue == "" && t.ConfigurationURL.DefaultCacheControl != "" {
		if req.Response.Header == nil {
			req.Response.Header = http.Header{}
		}
		req = req.WithContext(ctx.WithValue(req.Context(), context.CacheControlCtx, t.ConfigurationURL.DefaultCacheControl))
		req.Response.Header.Set("Cache-Control", t.ConfigurationURL.DefaultCacheControl)
	}

	cacheKey, cacheable, cachedResp := t.BaseRoundTrip(req)

	if cacheable && cachedResp != nil {
		rDate, _ := time.Parse(time.RFC1123, req.Header.Get("Date"))
		if cachedResp.Header == nil {
			cachedResp.Header = http.Header{}
		}
		cachedResp.Header.Set("Date", rDate.Format(http.TimeFormat))
	} else {
		if _, err := commonCacheControl(req, t, true); err != nil {
			return nil, err
		}
	}

	req.Response = cachedResp

	if cacheable && canStore(parseCacheControl(req.Header), parseCacheControl(req.Response.Header), req.Response.StatusCode) {
		_ = validateVary(req, req.Response, cacheKey, t)
	} else {
		MissCache(req.Response.Header.Set, req, "NOT-CACHEABLE")
	}

	return req.Response, nil
}

// RoundTrip takes a Request and returns a Response
//
// If there is a fresh Response already in cache, then it will be returned without connecting to
// the server.
//
// If there is a stale Response, then any validators it contains will be set on the new request
// to give the server a chance to respond with NotModified. If this happens, then the cached Response
// will be returned.
func (t *VaryTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	cacheKey, cacheable, resp := t.BaseRoundTrip(req)

	transport := t.Transport.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	if cacheable && resp != nil {
		r := commonVaryMatchesVerification(resp, req)
		if r != nil {
			return r, nil
		}

		if (err != nil || resp.StatusCode >= 500) &&
			req.Context().Value(context.SupportedMethod).(bool) && canStaleOnError(resp.Header, req.Header) {
			// In case of transport failure and stale-if-error activated, returns cached content
			// when available
			return resp, nil
		} else {
			if err != nil || resp.StatusCode != http.StatusOK {
				t.deleteCache(cacheKey)
			}
			if err != nil {
				return nil, err
			}
		}
	} else {
		if resp, err = commonCacheControl(req, transport, false); err != nil {
			return nil, err
		}
		MissCache(resp.Header.Set, req, "NOT-CACHEABLE-OR-NIL-RESPONSE")
	}
	req.Response = resp
	if !(cacheable && canStore(parseCacheControl(req.Header), parseCacheControl(resp.Header), req.Response.StatusCode) && validateVary(req, resp, cacheKey, t)) {
		go func() {
			t.Transport.CoalescingLayerStorage.Set(cacheKey)
		}()
		t.deleteCache(cacheKey)
	}
	return resp, nil
}
