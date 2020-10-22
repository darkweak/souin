package types

import (
	"github.com/darkweak/souin/configuration"
	"regexp"
)

type RetrieverResponsePropertiesInterface interface {
	GetProvider() AbstractProviderInterface
	GetConfiguration() configuration.AbstractConfigurationInterface
	GetMatchedURL() configuration.URL
	SetMatchedURL(url configuration.URL)
	GetRegexpUrls() *regexp.Regexp
}

type RetrieverResponseProperties struct {
	Provider AbstractProviderInterface
	Configuration configuration.AbstractConfigurationInterface
	MatchedURL configuration.URL
	RegexpUrls regexp.Regexp
}

func (r *RetrieverResponseProperties) GetProvider() AbstractProviderInterface {
	return r.Provider
}

func (r *RetrieverResponseProperties) GetConfiguration() configuration.AbstractConfigurationInterface {
	return r.Configuration
}

func (r *RetrieverResponseProperties) GetMatchedURL() configuration.URL {
	return r.MatchedURL
}

func (r *RetrieverResponseProperties) SetMatchedURL(url configuration.URL) {
	r.MatchedURL = url
}

func (r *RetrieverResponseProperties) GetRegexpUrls() *regexp.Regexp {
	return &r.RegexpUrls
}
