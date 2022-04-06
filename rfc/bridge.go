package rfc

import (
	"bufio"
	"bytes"
	"net/http"
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
func CachedResponse(c types.AbstractProviderInterface, req *http.Request, cachedKey string, transport types.TransportInterface, update bool) (*http.Response, error) {
	clonedReq := cloneRequest(req)
	cachedVal := c.Prefix(cachedKey, req)
	b := bytes.NewBuffer(cachedVal)
	response, _ := http.ReadResponse(bufio.NewReader(b), clonedReq)

	if update && nil != response && ValidateCacheControl(response) {
		SetCacheStatusEventually(response)
		go func() {
			clonedReq.Response = response
			// Update current cached response in background
			_, _ = transport.UpdateCacheEventually(clonedReq)
		}()
	}
	return response, nil
}

func commonCacheControl(req *http.Request, t http.RoundTripper) (*http.Response, error) {
	reqCacheControl := parseCacheControl(req.Header)
	if _, ok := reqCacheControl["only-if-cached"]; ok {
		return newGatewayTimeoutResponse(req), nil
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

// BaseRoundTrip is the base for RoundTrip
func (t *VaryTransport) BaseRoundTrip(req *http.Request, shouldReUpdate bool) (string, bool, *http.Response) {
	cacheKey := req.Context().Value(context.Key).(string)
	cacheable := IsVaryCacheable(req)
	cachedResp := req.Response
	if cachedResp == nil {
		cachedResp = new(http.Response)
	}

	if cachedResp.Header == nil {
		cachedResp.Header = make(http.Header)
	}

	if cacheable {
		cr, _ := CachedResponse(t.GetProvider(), req, cacheKey, t, shouldReUpdate)
		if cr != nil {
			cachedResp = cr
		}
	} else {
		go func() {
			t.CoalescingLayerStorage.Set(cacheKey)
		}()
		t.deleteCache(cacheKey)
	}

	return cacheKey, cacheable, cachedResp
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

// UpdateCacheEventually will handle Request and update the previous one in the cache provider
func (t *VaryTransport) UpdateCacheEventually(req *http.Request) (*http.Response, error) {
	if req.Response.Header.Get("Cache-Control") == "" && t.ConfigurationURL.DefaultCacheControl != "" {
		if req.Response.Header == nil {
			req.Response.Header = http.Header{}
		}
		req.Response.Header.Set("Cache-Control", t.ConfigurationURL.DefaultCacheControl)
	}

	cacheKey, cacheable, cachedResp := t.BaseRoundTrip(req, false)

	if cacheable && cachedResp != nil {
		rDate, _ := time.Parse(time.RFC1123, req.Header.Get("Date"))
		cachedResp.Header.Set("Date", rDate.Format(http.TimeFormat))
	} else {
		if _, err := commonCacheControl(req, t); err != nil {
			return nil, err
		}
	}

	req.Response = cachedResp

	if cacheable && canStore(parseCacheControl(req.Header), parseCacheControl(req.Response.Header), req.Response.StatusCode) {
		_ = validateVary(req, req.Response, cacheKey, t)
	} else {
		req.Response.Header.Set("Cache-Status", "Souin; fwd=uri-miss")
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
	cacheKey, cacheable, cachedResp := t.BaseRoundTrip(req, true)

	transport := t.Transport.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	if cacheable && cachedResp != nil {
		r := commonVaryMatchesVerification(cachedResp, req)
		if r != nil {
			return r, nil
		}

		var resp *http.Response
		resp, err = transport.RoundTrip(req)
		if (err != nil || resp.StatusCode >= 500) &&
			req.Context().Value(context.SupportedMethod).(bool) && canStaleOnError(cachedResp.Header, req.Header) {
			// In case of transport failure and stale-if-error activated, returns cached content
			// when available
			return cachedResp, nil
		} else {
			if err != nil || cachedResp.StatusCode != http.StatusOK {
				t.deleteCache(cacheKey)
			}
			if err != nil {
				return nil, err
			}
		}
	} else {
		if resp, err = commonCacheControl(req, transport); err != nil {
			return nil, err
		}
		resp.Header.Set("Cache-Status", "Souin; fwd=uri-miss")
	}
	resp, _ = transport.RoundTrip(req)
	if !(cacheable && canStore(parseCacheControl(req.Header), parseCacheControl(resp.Header), req.Response.StatusCode) && validateVary(req, resp, cacheKey, t)) {
		go func() {
			t.Transport.CoalescingLayerStorage.Set(cacheKey)
		}()
		t.deleteCache(cacheKey)
	}
	return resp, nil
}
