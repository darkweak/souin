package providers

import (
	"crypto/tls"
	"github.com/darkweak/souin/configuration"
	"testing"
)

func mockConfiguration() configuration.Configuration {
	return configuration.Configuration{
		SSLProviders:    []string{},
		ReverseProxyURL: "http://traefik",
		TTL:             "100",
		Cache: configuration.Cache{
			Headers:   []string{},
			Providers: []string{},
		},
		Regex: configuration.Regex{
			Exclude: "MyCustomRegex",
		},
		Redis: configuration.Redis{
			URL: "redis:6379",
		},
	}
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
	InitProviders(config, &configChannel, mockConfiguration())
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
}
