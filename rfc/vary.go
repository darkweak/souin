package rfc

import (
	"io"
	"io/ioutil"
	"net/http"
)

// varyMatches will return false unless all of the cached values for the headers listed in Vary
// match the new request
func varyMatches(cachedResp *http.Response, req *http.Request) bool {
	for _, header := range headerAllCommaSepValues(cachedResp.Header, "vary") {
		header = http.CanonicalHeaderKey(header)
		if header != "" && req.Header.Get(header) != cachedResp.Header.Get("X-Varied-"+header) {
			return false
		}
	}
	return true
}

func validateVary(req *http.Request, resp *http.Response, key string, t *VaryTransport) bool {
	if resp != nil && canStore(parseCacheControl(req.Header), parseCacheControl(resp.Header)) {
		variedHeaders := headerAllCommaSepValues(resp.Header, "vary")
		cacheKey := key
		if len(variedHeaders) > 0 {
			go func() {
				t.VaryLayerStorage.Set(key, variedHeaders)
			}()
			for _, varyKey := range variedHeaders {
				varyKey = http.CanonicalHeaderKey(varyKey)
				fakeHeader := "X-Varied-" + varyKey
				reqValue := req.Header.Get(varyKey)
				if reqValue != "" {
					resp.Header.Set(fakeHeader, reqValue)
				}
			}
			cacheKey = GetVariedCacheKey(req, variedHeaders)
		}
		switch req.Method {
		case http.MethodGet:
			// Delay caching until EOF is reached.
			resp.Body = &cachingReadCloser{
				R: resp.Body,
				OnEOF: func(r io.Reader) {
					resp := *resp
					resp.Body = ioutil.NopCloser(r)
					t.SetCache(cacheKey, &resp, req)
				},
			}
		}
		return true
	}

	return false
}
