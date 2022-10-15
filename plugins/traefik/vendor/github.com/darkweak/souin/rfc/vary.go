package rfc

import (
	"fmt"
	"net/http"

	"github.com/darkweak/souin/context"
)

// varyMatches will return false unless all of the cached values for the headers listed in Vary
// match the new request
func varyMatches(cachedResp *http.Response, req *http.Request) bool {
	for _, header := range headerAllCommaSepValues(cachedResp.Header) {
		header = http.CanonicalHeaderKey(header)
		if header == "" || req.Header.Get(header) == "" {
			return false
		}
	}
	return true
}

func validateVary(req *http.Request, resp *http.Response, key string, t *VaryTransport) bool {
	if resp != nil {
		variedHeaders := headerAllCommaSepValues(resp.Header)
		cacheKey := key
		if len(variedHeaders) > 0 {
			cacheKey = GetVariedCacheKey(req, variedHeaders)
		}
		resp.Header.Del("Age")
		status := fmt.Sprintf("%s; fwd=uri-miss", req.Context().Value(context.CacheName))
		if t.SetCache(cacheKey, resp, req.Context().Value(context.CacheControlCtx).(string)) {
			status += "; stored"
			_ = t.SurrogateStorage.Store(resp, cacheKey)
		}
		resp.Header.Set("Cache-Status", status)
		go func() {
			t.CoalescingLayerStorage.Delete(cacheKey)
		}()
		return true
	}

	return false
}
