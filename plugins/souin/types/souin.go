package types

import (
	"net/url"
	"regexp"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage/types"
)

// SouinRetrieverResponseProperties struct
type SouinRetrieverResponseProperties struct {
	RetrieverResponseProperties
	ReverseProxyURL *url.URL
}

// GetProvider interface
func (r *SouinRetrieverResponseProperties) GetProvider() types.Storer {
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
func (r *SouinRetrieverResponseProperties) GetTransport() TransportInterface {
	return r.RetrieverResponseProperties.Transport
}

// SetTransport set the transport
func (r *SouinRetrieverResponseProperties) SetTransport(transportInterface TransportInterface) {
	r.RetrieverResponseProperties.Transport = transportInterface
}
