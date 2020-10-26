package types

import (
	configuration_types "github.com/darkweak/souin/configuration_types"
	"regexp"
)

// RetrieverResponsePropertiesInterface interface
type RetrieverResponsePropertiesInterface interface {
	GetProvider() AbstractProviderInterface
	GetConfiguration() configuration_types.AbstractConfigurationInterface
	GetMatchedURL() configuration_types.URL
	SetMatchedURL(url configuration_types.URL)
	GetRegexpUrls() *regexp.Regexp
}

// RetrieverResponseProperties struct
type RetrieverResponseProperties struct {
	Provider      AbstractProviderInterface
	Configuration configuration_types.AbstractConfigurationInterface
	MatchedURL    configuration_types.URL
	RegexpUrls    regexp.Regexp
}

// GetProvider interface
func (r *RetrieverResponseProperties) GetProvider() AbstractProviderInterface {
	return r.Provider
}

// GetConfiguration get the configuration
func (r *RetrieverResponseProperties) GetConfiguration() configuration_types.AbstractConfigurationInterface {
	return r.Configuration
}

// GetMatchedURL get the matched url
func (r *RetrieverResponseProperties) GetMatchedURL() configuration_types.URL {
	return r.MatchedURL
}

// SetMatchedURL set the matched url
func (r *RetrieverResponseProperties) SetMatchedURL(url configuration_types.URL) {
	r.MatchedURL = url
}

// GetRegexpUrls get the regexp urls
func (r *RetrieverResponseProperties) GetRegexpUrls() *regexp.Regexp {
	return &r.RegexpUrls
}
