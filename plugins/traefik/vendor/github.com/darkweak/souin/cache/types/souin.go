package types

import (
	"github.com/darkweak/souin/cache/surrogate/providers"
	"net/http"
	"net/http/httputil"
	"regexp"
	"time"

	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
)

// TransportInterface interface
type TransportInterface interface {
	GetProvider() AbstractProviderInterface
	RoundTrip(req *http.Request) (resp *http.Response, err error)
	SetURL(url configurationtypes.URL)
	UpdateCacheEventually(req *http.Request) (resp *http.Response, err error)
	GetCoalescingLayerStorage() *CoalescingLayerStorage
	GetYkeyStorage() *ykeys.YKeyStorage
	GetSurrogateKeys() providers.SurrogateInterface
}

// Transport is an implementation of http.RoundTripper that will return values from a cache
// where possible (avoiding a network request) and will additionally add validators (etag/if-modified-since)
// to repeated requests allowing servers to return 304 / Not Modified
type Transport struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport              http.RoundTripper
	Provider               AbstractProviderInterface
	ConfigurationURL       configurationtypes.URL
	MarkCachedResponses    bool
	CoalescingLayerStorage *CoalescingLayerStorage
	YkeyStorage            *ykeys.YKeyStorage
	SurrogateStorage       providers.SurrogateInterface
}

// GetProvider returns the associated provider
func (t *Transport) GetProvider() AbstractProviderInterface {
	return t.Provider
}

// SetURL set the URL
func (t *Transport) SetURL(url configurationtypes.URL) {
	t.ConfigurationURL = url
}

// GetCoalescingLayerStorage get the coalescing layer storage
func (t *Transport) GetCoalescingLayerStorage() *CoalescingLayerStorage {
	return t.CoalescingLayerStorage
}

// GetYkeyStorage get the ykeys storage
func (t *Transport) GetYkeyStorage() *ykeys.YKeyStorage {
	return t.YkeyStorage
}

// GetSurrogateKeys get the surrogate keys storage
func (t *Transport) GetSurrogateKeys() providers.SurrogateInterface {
	return t.SurrogateStorage
}

// SetCache set the cache
func (t *Transport) SetCache(key string, resp *http.Response) {
	if respBytes, err := httputil.DumpResponse(resp, true); err == nil {
		go func() {
			if t.YkeyStorage != nil {
				t.YkeyStorage.AddToTags(key, t.YkeyStorage.GetValidatedTags(key, resp.Header))
			}
		}()
		t.Provider.Set(key, respBytes, t.ConfigurationURL, time.Duration(0))
	}
}

// RetrieverResponsePropertiesInterface interface
type RetrieverResponsePropertiesInterface interface {
	GetProvider() AbstractProviderInterface
	GetConfiguration() configurationtypes.AbstractConfigurationInterface
	GetMatchedURL() configurationtypes.URL
	SetMatchedURL(url configurationtypes.URL)
	GetRegexpUrls() *regexp.Regexp
	GetTransport() TransportInterface
	SetTransport(TransportInterface)
	GetExcludeRegexp() *regexp.Regexp
}

// RetrieverResponseProperties struct
type RetrieverResponseProperties struct {
	Provider      AbstractProviderInterface
	Configuration configurationtypes.AbstractConfigurationInterface
	MatchedURL    configurationtypes.URL
	RegexpUrls    regexp.Regexp
	Transport     TransportInterface
	ExcludeRegex  *regexp.Regexp
}

// GetProvider interface
func (r *RetrieverResponseProperties) GetProvider() AbstractProviderInterface {
	return r.Provider
}

// GetConfiguration get the configuration
func (r *RetrieverResponseProperties) GetConfiguration() configurationtypes.AbstractConfigurationInterface {
	return r.Configuration
}

// GetMatchedURL get the matched url
func (r *RetrieverResponseProperties) GetMatchedURL() configurationtypes.URL {
	return r.MatchedURL
}

// SetMatchedURL set the matched url
func (r *RetrieverResponseProperties) SetMatchedURL(url configurationtypes.URL) {
	r.MatchedURL = url
}

// GetRegexpUrls get the regexp urls
func (r *RetrieverResponseProperties) GetRegexpUrls() *regexp.Regexp {
	return &r.RegexpUrls
}

// GetTransport get the transport according to the RFC
func (r *RetrieverResponseProperties) GetTransport() TransportInterface {
	return r.Transport
}

// SetTransport set the transport
func (r *RetrieverResponseProperties) SetTransport(transportInterface TransportInterface) {
	r.Transport = transportInterface
}

// GetExcludeRegexp get the excluded regexp
func (r *RetrieverResponseProperties) GetExcludeRegexp() *regexp.Regexp {
	return r.ExcludeRegex
}
