package tests

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/configurationtypes"
	"log"
	"net/http"
	"regexp"
)

// DOMAIN is the domain constant
const DOMAIN = "domain.com"

// PATH is the path constant
const PATH = "/testing"

// BaseConfiguration is the legacy configuration
func BaseConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50
    headers:
      - Authorization
      - 'Content-Type'
`
}

// OlricConfiguration is the olric included configuration
func OlricConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  distributed: true
  headers:
    - Authorization
  olric:
    url: 'olric:3320'
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50
    headers:
      - Authorization
      - 'Content-Type'
`
}

// MockConfiguration is an helper to mock the configuration
func MockConfiguration(configurationToLoad func() string) *configuration.Configuration {
	var config configuration.Configuration
	e := config.Parse([]byte(configurationToLoad()))
	if e != nil {
		log.Fatal(e)
	}
	return &config
}

// MockInitializeRegexp is an helper to mock the regexp initialization
func MockInitializeRegexp(configurationInstance configurationtypes.AbstractConfigurationInterface) regexp.Regexp {
	u := ""
	for k := range configurationInstance.GetUrls() {
		if "" != u {
			u += "|"
		}
		u += "(" + k + ")"
	}

	return *regexp.MustCompile(u)
}

// GetTokenName returns the token name
func GetTokenName() string {
	return "souin-authorization-token"
}

// GetValidToken returns a valid token
func GetValidToken() *http.Cookie {
	return &http.Cookie{
		Name:  GetTokenName(),
		Value: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InVzZXIxIiwiZXhwIjoxNjE0MTI0Nzk5OX0.7blW8hKWls2UgHLU8KOzwTG13uNoJR3UhLgoVdyCzx0",
		Path:  "/",
	}
}

// GetCacheProviderClientAndMatchedURL will work as a factory to build providers from configuration and get the URL from the key passed in parameter
func GetCacheProviderClientAndMatchedURL(key string, configurationMocker func() configurationtypes.AbstractConfigurationInterface, factory func(configurationInterface configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error)) (types.AbstractProviderInterface, configurationtypes.URL) {
	config := configurationMocker()
	client, _ := factory(config)
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configurationtypes.URL{
		TTL:     config.GetDefaultCache().GetTTL(),
		Headers: config.GetDefaultCache().GetHeaders(),
	}
	if "" != regexpURL {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return client, matchedURL
}
