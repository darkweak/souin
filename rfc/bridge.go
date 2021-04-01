package rfc

import (
	"bufio"
	"bytes"
	"github.com/darkweak/souin/cache/types"
	"net/http"
)

const (
	stale = iota
	fresh
	transparent
	// XFromCache header constant
	XFromCache = "X-From-Cache"
)

// CachedResponse returns the cached http.Response for req if present, and nil
// otherwise.
func CachedResponse(c types.AbstractProviderInterface, req *http.Request, cachedKey string, transport types.TransportInterface, update bool) (types.ReverseResponse, error) {
	clonedReq := cloneRequest(req)
	cachedVal := c.Get(cachedKey)
	b := bytes.NewBuffer(cachedVal)
	response, _ := http.ReadResponse(bufio.NewReader(b), clonedReq)
	if update && nil != response {
		go func() {
			// Update current cached response in background
			_, _ = transport.UpdateCacheEventually(clonedReq)
		}()
	}
	return types.ReverseResponse{
		Response: response,
	}, nil
}

func (t *VaryTransport) BaseRoundTrip(req *http.Request, shouldReUpdate bool) (string, bool, *http.Response) {
	cacheKey := GetCacheKey(req)
	cacheable := IsVaryCacheable(req)
	cachedResp := req.Response
	if cacheable {
		varied := t.GetVaryLayerStorage().Get(cacheKey)
		if len(varied) != 0 {
			cacheKey = GetVariedCacheKey(req, varied)
		}
		cr, _ := CachedResponse(t.GetProvider(), req, cacheKey, t, shouldReUpdate)
		if cr.Response != nil {
			cachedResp = cr.Response
		}
	} else {
		t.Provider.Delete(cacheKey)
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
				req = req2
			}
		}
	}

	return nil
}

// UpdateCacheEventually will handle Request and update the previous one in the cache provider
func (t *VaryTransport) UpdateCacheEventually(req *http.Request) (resp *http.Response, err error) {
	cacheKey, cacheable, cachedResp := t.BaseRoundTrip(req, false)

	if cacheable && cachedResp != nil {
		r := commonVaryMatchesVerification(cachedResp, req)
		if r != nil {
			return r, nil
		}
	} else {
		reqCacheControl := parseCacheControl(req.Header)
		if _, ok := reqCacheControl["only-if-cached"]; ok {
			resp = newGatewayTimeoutResponse(req)
		} else {
			resp, err = t.RoundTrip(req)
			if err != nil {
				return nil, err
			}
		}
	}

	resp = cachedResp
	if cacheable {
		_ = validateVary(req, resp, cacheKey, t)
	}

	return resp, nil
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

	transport := t.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	if cacheable && cachedResp != nil {
		if t.MarkCachedResponses {
			cachedResp.Header.Set(XFromCache, "1")
		}

		r := commonVaryMatchesVerification(cachedResp, req)
		if r != nil {
			return r, nil
		}

		resp, err = transport.RoundTrip(req)
		if err == nil && req.Method == http.MethodGet && resp.StatusCode == http.StatusNotModified {
			// Replace the 304 response with the one from cache, but update with some new headers
			endToEndHeaders := getEndToEndHeaders(resp.Header)
			for _, header := range endToEndHeaders {
				cachedResp.Header[header] = resp.Header[header]
			}
			resp = cachedResp
		} else if (err != nil || resp.StatusCode >= 500) &&
			req.Method == http.MethodGet && canStaleOnError(cachedResp.Header, req.Header) {
			// In case of transport failure and stale-if-error activated, returns cached content
			// when available
			return cachedResp, nil
		} else {
			if err != nil || cachedResp.StatusCode != http.StatusOK {
				t.Provider.Delete(cacheKey)
			}
			if err != nil {
				return nil, err
			}
		}
	} else {
		reqCacheControl := parseCacheControl(req.Header)
		if _, ok := reqCacheControl["only-if-cached"]; ok {
			resp = newGatewayTimeoutResponse(req)
		} else {
			resp, err = transport.RoundTrip(req)
			if err != nil {
				return nil, err
			}
		}
	}
	resp, err = transport.RoundTrip(req)
	if !(cacheable && validateVary(req, resp, cacheKey, t)) {
		go func() {
			t.CoalescingLayerStorage.Set(cacheKey)
		}()
		t.Provider.Delete(cacheKey)
	}
	return resp, nil
}
