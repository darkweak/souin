package rfc

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
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
func NewTransport(p map[string]types.AbstractProviderInterface) *VaryTransport {
	return &VaryTransport{
		Providers:              p,
		VaryLayerStorage:       types.InitializeVaryLayerStorage(),
		CoalescingLayerStorage: types.InitializeCoalescingLayerStorage(),
		MarkCachedResponses:    true,
	}
}

// GetProviders returns the associated provider
func (t *VaryTransport) GetProviders() map[string]types.AbstractProviderInterface {
	return t.Providers
}

// SetURL set the URL
func (t *VaryTransport) SetURL(url configurationtypes.URL) {
	t.ConfigurationURL = url
}

// GetVaryLayerStorage get the vary layer storagecache/coalescing/requestCoalescing_test.go
func (t *VaryTransport) GetVaryLayerStorage() *types.VaryLayerStorage {
	return t.VaryLayerStorage
}

// GetCoalescingLayerStorage get the coalescing layer storage
func (t *VaryTransport) GetCoalescingLayerStorage() *types.CoalescingLayerStorage {
	return t.CoalescingLayerStorage
}

// SetCache set the cache
func (t *VaryTransport) SetCache(key string, resp *http.Response) {
	if respBytes, err := httputil.DumpResponse(resp, true); err == nil {
		for _, p := range t.Providers {
			p.Set(key, respBytes, t.ConfigurationURL, time.Duration(0))
		}
	}
}
