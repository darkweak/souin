package rfc

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/darkweak/souin/cache/surrogate/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/context"
	"github.com/pquerna/cachecontrol/cacheobject"
)

// VaryTransport type
type VaryTransport struct {
	*types.Transport
}

// IsVaryCacheable determines if it's cacheable
func IsVaryCacheable(req *http.Request) bool {
	return req.Context().Value(context.SupportedMethod).(bool) && req.Header.Get("range") == ""
}

// NewTransport returns a new Transport with the
// provided Cache implementation and MarkCachedResponses set to true
func NewTransport(p types.AbstractProviderInterface, ykeyStorage *ykeys.YKeyStorage, surrogateStorage providers.SurrogateInterface) *VaryTransport {
	return &VaryTransport{
		&types.Transport{
			Provider:               p,
			CoalescingLayerStorage: types.InitializeCoalescingLayerStorage(),
			MarkCachedResponses:    true,
			YkeyStorage:            ykeyStorage,
			SurrogateStorage:       surrogateStorage,
		},
	}
}

// GetProvider returns the associated provider
func (t *VaryTransport) GetProvider() types.AbstractProviderInterface {
	return t.Transport.Provider
}

// SetURL set the URL
func (t *VaryTransport) SetURL(url configurationtypes.URL) {
	t.Transport.ConfigurationURL = url
}

// GetCoalescingLayerStorage get the coalescing layer storage
func (t *VaryTransport) GetCoalescingLayerStorage() *types.CoalescingLayerStorage {
	return t.Transport.CoalescingLayerStorage
}

// GetYkeyStorage get the ykeys storage
func (t *VaryTransport) GetYkeyStorage() *ykeys.YKeyStorage {
	return t.Transport.YkeyStorage
}

// GetSurrogateKeys get the surrogate keys storage
func (t *VaryTransport) GetSurrogateKeys() providers.SurrogateInterface {
	return t.Transport.SurrogateStorage
}

// SetSurrogateKeys set the surrogate keys storage
func (t *VaryTransport) SetSurrogateKeys(s providers.SurrogateInterface) {
	t.Transport.SurrogateStorage = s
}

// SetCache set the cache
func (t *VaryTransport) SetCache(key string, resp *http.Response, ccValue string) {
	co, e := cacheobject.ParseResponseCacheControl(ccValue)
	if e != nil {
		return
	}
	ma := t.ConfigurationURL.TTL.Duration
	if co.MaxAge > 0 {
		ma = time.Duration(co.MaxAge) * time.Second
	} else if co.SMaxAge > 0 {
		ma = time.Duration(co.SMaxAge) * time.Second
	}
	if ma > t.ConfigurationURL.TTL.Duration {
		ma = t.ConfigurationURL.TTL.Duration
	}
	date, _ := http.ParseTime(resp.Header.Get("Date"))
	resp.Header.Set(storedTTLHeader, ma.String())
	ma = ma - time.Since(date)
	if respBytes, err := httputil.DumpResponse(resp, true); err == nil {
		t.Provider.Set(key, respBytes, t.ConfigurationURL, ma)
	}
}
