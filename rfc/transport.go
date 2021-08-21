package rfc

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
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
func NewTransport(p types.AbstractProviderInterface, ykeyStorage *ykeys.YKeyStorage) *VaryTransport {
	return &VaryTransport{
		Provider:               p,
		CoalescingLayerStorage: types.InitializeCoalescingLayerStorage(),
		MarkCachedResponses:    true,
		YkeyStorage:            ykeyStorage,
	}
}

// GetProvider returns the associated provider
func (t *VaryTransport) GetProvider() types.AbstractProviderInterface {
	return t.Provider
}

// SetURL set the URL
func (t *VaryTransport) SetURL(url configurationtypes.URL) {
	t.ConfigurationURL = url
}

// GetCoalescingLayerStorage get the coalescing layer storage
func (t *VaryTransport) GetCoalescingLayerStorage() *types.CoalescingLayerStorage {
	return t.CoalescingLayerStorage
}

// GetYkeyStorage get the ykeys storage
func (t *VaryTransport) GetYkeyStorage() *ykeys.YKeyStorage {
	return t.YkeyStorage
}

// SetCache set the cache
func (t *VaryTransport) SetCache(key string, resp *http.Response) {
	if respBytes, err := httputil.DumpResponse(resp, true); err == nil {
		go func() {
			t.YkeyStorage.AddToTags(key, t.YkeyStorage.GetValidatedTags(key, resp.Header))
		}()
		t.Provider.Set(key, respBytes, t.ConfigurationURL, time.Duration(0))
	}
}
