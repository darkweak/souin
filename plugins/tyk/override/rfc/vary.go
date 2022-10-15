package rfc

import (
	"fmt"
	"io"
	"io/ioutil"
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
		// Delay caching until EOF is reached.
		resp.Body = &cachingReadCloser{
			R: resp.Body,
			OnEOF: func(r io.Reader) {
				re := *resp
				re.Body = ioutil.NopCloser(r)
				_ = t.SurrogateStorage.Store(&re, cacheKey)
				status := fmt.Sprintf("%s; fwd=uri-miss", req.Context().Value(context.CacheName))
				if t.SetCache(cacheKey, resp, req.Context().Value(context.CacheControlCtx).(string)) {
					status += "; stored"
					_ = t.SurrogateStorage.Store(resp, cacheKey)
				}
				resp.Header.Set("Cache-Status", status)
				go func() {
					t.CoalescingLayerStorage.Delete(cacheKey)
				}()
			},
		}
		return true
	}

	return false
}
