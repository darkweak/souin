package types

import (
	"github.com/darkweak/souin/configurationtypes"
	"net/url"
	"regexp"
)

// RetrieverResponsePropertiesInterface interface
type RetrieverResponsePropertiesInterface interface {
	GetProvider() AbstractProviderInterface
	GetConfiguration() configurationtypes.AbstractConfigurationInterface
	GetMatchedURL() configurationtypes.URL
	SetMatchedURL(url configurationtypes.URL)
	GetRegexpUrls() *regexp.Regexp
	GetReverseProxyURL() *url.URL
}

// RetrieverResponseProperties struct
type RetrieverResponseProperties struct {
	Provider        AbstractProviderInterface
	Configuration   configurationtypes.AbstractConfigurationInterface
	MatchedURL      configurationtypes.URL
	RegexpUrls      regexp.Regexp
	ReverseProxyURL *url.URL
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

// GetReverseProxyURL get the reverse proxy url
func (r *RetrieverResponseProperties) GetReverseProxyURL() *url.URL {
	return r.ReverseProxyURL
}
