package providers

import (
	"testing"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/configuration_types"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/helpers"
	"regexp"
	"log"
)

func MockConfiguration() configuration_types.AbstractConfigurationInterface {
	var config configuration.Configuration
	e := config.Parse([]byte(`
default_cache:
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
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

func MockInitializeRegexp(configurationInstance configuration_types.AbstractConfigurationInterface) regexp.Regexp {
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
	c := MockConfiguration()
	p := InitializeProvider(c)
	err := p.Init()
	if nil != err {
		errors.GenerateError(t, "Init shouldn't crash")
	}
}

func TestPathnameNotInExcludeRegex(t *testing.T) {
	config := MockConfiguration()
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
