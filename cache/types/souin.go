package types

import (
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"net/url"
	"regexp"
)

// TransportInterface interface
type TransportInterface interface {
	GetProviders() map[string]AbstractProviderInterface
	RoundTrip(req *http.Request) (resp *http.Response, err error)
	SetURL(url configurationtypes.URL)
	UpdateCacheEventually(req *http.Request) (resp *http.Response, err error)
}

// Transport is an implementation of http.RoundTripper that will return values from a cache
// where possible (avoiding a network request) and will additionally add validators (etag/if-modified-since)
// to repeated requests allowing servers to return 304 / Not Modified
type Transport struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport        http.RoundTripper
	Providers        map[string]AbstractProviderInterface
	ConfigurationURL configurationtypes.URL
	// If true, responses returned from the cache will be given an extra header, X-From-Cache
	MarkCachedResponses bool
}

// RetrieverResponsePropertiesInterface interface
type RetrieverResponsePropertiesInterface interface {
	GetProviders() map[string]AbstractProviderInterface
	GetConfiguration() configurationtypes.AbstractConfigurationInterface
	GetMatchedURL() configurationtypes.URL
	SetMatchedURL(url configurationtypes.URL)
	GetRegexpUrls() *regexp.Regexp
	GetReverseProxyURL() *url.URL
	GetTransport() TransportInterface
	SetTransport(TransportInterface)
}

// RetrieverResponseProperties struct
type RetrieverResponseProperties struct {
	Providers       map[string]AbstractProviderInterface
	Configuration   configurationtypes.AbstractConfigurationInterface
	MatchedURL      configurationtypes.URL
	RegexpUrls      regexp.Regexp
	ReverseProxyURL *url.URL
	Transport       TransportInterface
}

// GetProviders interface
func (r *RetrieverResponseProperties) GetProviders() map[string]AbstractProviderInterface {
	return r.Providers
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
	providers := url.Providers
	if nil == providers || 0 == len(providers) {
		for k := range r.Providers {
			providers = append(providers, k)
		}
	}
	url.Providers = providers
	r.MatchedURL = url
}

// GetRegexpUrls get the regexp urls
func (r *RetrieverResponseProperties) GetRegexpUrls() *regexp.Regexp {
	return &r.RegexpUrls
}

// GetReverseProxyURL get the reverse proxy url
func (r *RetrieverResponseProperties) GetReverseProxyURL() *url.URL {
	return r.ReverseProxyURL
}

// GetTransport get the transport according to the RFC
func (r *RetrieverResponseProperties) GetTransport() TransportInterface {
	return r.Transport
}

// SetTransport set the transport
func (r *RetrieverResponseProperties) SetTransport(transportInterface TransportInterface) {
	r.Transport = transportInterface
}
