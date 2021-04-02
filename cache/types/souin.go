package types

import (
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"regexp"
)

// TransportInterface interface
type TransportInterface interface {
	GetProvider() AbstractProviderInterface
	RoundTrip(req *http.Request) (resp *http.Response, err error)
	SetURL(url configurationtypes.URL)
	UpdateCacheEventually(req *http.Request) (resp *http.Response, err error)
	GetVaryLayerStorage() *VaryLayerStorage
	GetCoalescingLayerStorage() *CoalescingLayerStorage
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
	VaryLayerStorage       *VaryLayerStorage
	CoalescingLayerStorage *CoalescingLayerStorage
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
}

// RetrieverResponseProperties struct
type RetrieverResponseProperties struct {
	Provider      AbstractProviderInterface
	Configuration configurationtypes.AbstractConfigurationInterface
	MatchedURL    configurationtypes.URL
	RegexpUrls    regexp.Regexp
	Transport     TransportInterface
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
