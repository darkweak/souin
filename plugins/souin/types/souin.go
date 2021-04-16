package types

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"net/url"
	"regexp"
)

// SouinRetrieverResponseProperties struct
type SouinRetrieverResponseProperties struct {
	types.RetrieverResponseProperties
	ReverseProxyURL *url.URL
}

// GetProviders interface
func (r *SouinRetrieverResponseProperties) GetProviders() map[string]types.AbstractProviderInterface {
	return r.Providers
}

// GetConfiguration get the configuration
func (r *SouinRetrieverResponseProperties) GetConfiguration() configurationtypes.AbstractConfigurationInterface {
	return r.Configuration
}

// GetMatchedURL get the matched url
func (r *SouinRetrieverResponseProperties) GetMatchedURL() configurationtypes.URL {
	return r.MatchedURL
}

// SetMatchedURL set the matched url
func (r *SouinRetrieverResponseProperties) SetMatchedURL(url configurationtypes.URL) {
	r.MatchedURL = url
}

// GetRegexpUrls get the regexp urls
func (r *SouinRetrieverResponseProperties) GetRegexpUrls() *regexp.Regexp {
	return &r.RegexpUrls
}

// GetReverseProxyURL get the reverse proxy url
func (r *SouinRetrieverResponseProperties) GetReverseProxyURL() *url.URL {
	return r.ReverseProxyURL
}

// GetTransport get the transport according to the RFC
func (r *SouinRetrieverResponseProperties) GetTransport() types.TransportInterface {
	return r.Transport
}

// SetTransport set the transport
func (r *SouinRetrieverResponseProperties) SetTransport(transportInterface types.TransportInterface) {
	r.Transport = transportInterface
}
