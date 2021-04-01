package rfc

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/dgraph-io/ristretto"
	"github.com/pquerna/cachecontrol"
	"net/http"
	"net/http/httputil"
	"time"
)

// VaryTransport type
type VaryTransport types.Transport

// IsVaryCacheable determines if it's cacheable
func IsVaryCacheable(req *http.Request) bool {
	method := req.Method
	rangeHeader := req.Header.Get("range")
	return (method == http.MethodGet || method == http.MethodHead) && rangeHeader == ""
}

// NewTransport returns a new Transport with the
// provided Cache implementation and MarkCachedResponses set to true
func NewTransport(p types.AbstractProviderInterface) *VaryTransport {
	storage, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
		OnEvict: func(key, conflict uint64, value interface{}, cost int64) {},
	})
	return &VaryTransport{Provider: p, VaryLayerStorage: &types.VaryLayerStorage{Cache: storage}, MarkCachedResponses: true}
}

// GetProvider returns the associated provider
func (t *VaryTransport) GetProvider() types.AbstractProviderInterface {
	return t.Provider
}

// SetURL set the URL
func (t *VaryTransport) SetURL(url configurationtypes.URL) {
	t.ConfigurationURL = url
}

func (t *VaryTransport) GetVaryLayerStorage() *types.VaryLayerStorage {
	return t.VaryLayerStorage
}

// SetCache set the cache
func (t *VaryTransport) SetCache(key string, resp *http.Response, req *http.Request) {
	r, _, _ := cachecontrol.CachableResponse(req, resp, cachecontrol.Options{})
	resp.Header.Set(XFromCache, "Souin")
	respBytes, err := httputil.DumpResponse(resp, true)
	if err == nil && len(r) == 0 {
		t.Provider.Set(key, respBytes, t.ConfigurationURL, time.Duration(0))
	}
}
