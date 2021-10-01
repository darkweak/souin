package types

import (
	"net/url"
	"regexp"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// SouinRetrieverResponseProperties struct
type SouinRetrieverResponseProperties struct {
	types.RetrieverResponseProperties
	ReverseProxyURL *url.URL
}

// GetProvider interface
func (r *SouinRetrieverResponseProperties) GetProvider() types.AbstractProviderInterface {
	return r.RetrieverResponseProperties.Provider
}

// GetConfiguration get the configuration
func (r *SouinRetrieverResponseProperties) GetConfiguration() configurationtypes.AbstractConfigurationInterface {
	return r.RetrieverResponseProperties.Configuration
}

// GetMatchedURL get the matched url
func (r *SouinRetrieverResponseProperties) GetMatchedURL() configurationtypes.URL {
	return r.RetrieverResponseProperties.MatchedURL
}

// SetMatchedURL set the matched url
func (r *SouinRetrieverResponseProperties) SetMatchedURL(url configurationtypes.URL) {
	r.RetrieverResponseProperties.MatchedURL = url
}

// GetRegexpUrls get the regexp urls
func (r *SouinRetrieverResponseProperties) GetRegexpUrls() *regexp.Regexp {
	return &r.RetrieverResponseProperties.RegexpUrls
}

// GetReverseProxyURL get the reverse proxy url
func (r *SouinRetrieverResponseProperties) GetReverseProxyURL() *url.URL {
	return r.ReverseProxyURL
}

// GetTransport get the transport according to the RFC
func (r *SouinRetrieverResponseProperties) GetTransport() types.TransportInterface {
	return r.RetrieverResponseProperties.Transport
}

// SetTransport set the transport
func (r *SouinRetrieverResponseProperties) SetTransport(transportInterface types.TransportInterface) {
	r.RetrieverResponseProperties.Transport = transportInterface
}
