package coalescing

import (
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

const DOMAIN = "domain.com"
const PATH = "/testing"

func MockConfiguration() configurationtypes.AbstractConfigurationInterface {
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

func getMatchedURL(key string) configurationtypes.URL {
	config := MockConfiguration()
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configurationtypes.URL{
		TTL:     config.GetDefaultCache().TTL,
		Headers: config.GetDefaultCache().Headers,
	}
	if "" != regexpURL {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return matchedURL
}

func TestServeResponse(t *testing.T) {
	c := MockConfiguration()
	prs := providers.InitializeProvider(c)
	rc := Initialize()
	regexpUrls := helpers.InitializeRegexp(c)
	retriever := &types.RetrieverResponseProperties{
		Configuration: c,
		Provider:      prs,
		MatchedURL:    getMatchedURL(PATH),
		RegexpUrls:    regexpUrls,
	}
	r := httptest.NewRequest("GET", "http://"+DOMAIN+PATH, nil)
	w := httptest.NewRecorder()
	ServeResponse(
		w,
		r,
		retriever,
		func(rw http.ResponseWriter, rq *http.Request, r types.RetrieverResponsePropertiesInterface, rc RequestCoalescingInterface){},
		rc,
	)
}
