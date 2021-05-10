package tests

import (
	"fmt"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/configurationtypes"
	"io/ioutil"
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

// EmbeddedOlricConfiguration is the olric included configuration
func EmbeddedOlricConfiguration() string {
	path := "/tmp/olric.yml"
	_ = ioutil.WriteFile(
		path,
		[]byte(
			`
olricd:
  bindAddr: "0.0.0.0"
  bindPort: 3320
  serializer: "msgpack"
  keepAlivePeriod: "300s"
  bootstrapTimeout: "5s"
  partitionCount:  271
  replicaCount: 2
  writeQuorum: 1
  readQuorum: 1
  readRepair: false
  replicationMode: 1 # sync mode. for async, set 1
  tableSize: 1048576 # 1MB in bytes
  memberCountQuorum: 1

client:
  dialTimeout: "-1s"
  readTimeout: "3s"
  writeTimeout: "3s"
  keepAlive: "15s"
  minConn: 1
  maxConn: 100

logging:
  verbosity: 6
  level: "DEBUG"
  output: "stderr"

memberlist:
  environment: "local"
  bindAddr: "0.0.0.0"
  bindPort: 3322
  enableCompression: false
  joinRetryInterval: "1s"
  maxJoinAttempts: 10
`),
		0644,
)

	return fmt.Sprintf(`
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
  olric:
    path: '%s'
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
`, path)
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
