package providers

import (
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/tests"
	"log"
	"regexp"
	"testing"
)

func MockConfiguration() configurationtypes.AbstractConfigurationInterface {
	var config configuration.Configuration
	e := config.Parse([]byte(`
default_cache:
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  redis:
    url: 'redis:6379'
  regex:
    exclude: 'ARegexHere'
  ttl: 1000
reverse_proxy_url: 'http://traefik'
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
`))
	if e != nil {
		log.Fatal(e)
	}
	return &config
}

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

func TestInitializeProvider(t *testing.T) {
	c := tests.MockConfiguration()
	ps := InitializeProvider(c)
	for _, p := range ps {
		err := p.Init()
		if nil != err {
			errors.GenerateError(t, "Init shouldn't crash")
		}
	}
}

func TestPathnameNotInExcludeRegex(t *testing.T) {
	config := tests.MockConfiguration()
	if helpers.PathnameNotInExcludeRegex(config.GetDefaultCache().Regex.Exclude, config) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if helpers.PathnameNotInExcludeRegex(config.GetDefaultCache().Regex.Exclude+"/A", config) {
		errors.GenerateError(t, "Pathname should be in regex")
	}
	if !helpers.PathnameNotInExcludeRegex("/BadPath", config) {
		errors.GenerateError(t, "Pathname shouldn't be in regex")
	}
}
