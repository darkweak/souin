package providers

import (
	"crypto/tls"
	"testing"
	"github.com/darkweak/souin/errors"
	"fmt"
	"log"
	"github.com/darkweak/souin/configuration"
)

func MockConfiguration() configuration.AbstractConfigurationInterface {
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

func TestInitProviders(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		NameToCertificate:  make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config.Certificates = append(config.Certificates, v)
	InitProviders(config, &configChannel, MockConfiguration())
}

func TestLoadFromConfigFile(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		NameToCertificate:  make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config.Certificates = append(config.Certificates, v)

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 1 != len(config.Certificates) {
		errors.GenerateError(t, fmt.Sprintf("Certificates length %d not corresponding to received %d", 0, len(config.Certificates)))
	}
}

func TestLoadFromConfigFile_NoAdditionalProviders(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		NameToCertificate:  make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config.Certificates = append(config.Certificates, v)

	var providers []CommonProvider
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 1 != len(config.Certificates) {
		errors.GenerateError(t, fmt.Sprintf("Certificates length %d not corresponding to received %d", 0, len(config.Certificates)))
	}
}
