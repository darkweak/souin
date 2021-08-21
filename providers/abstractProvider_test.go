package providers

import (
	"crypto/tls"
	"fmt"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"testing"
)

func TestInitProviders(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config.Certificates = append(config.Certificates, v)
	InitProviders(config, &configChannel, tests.MockConfiguration(tests.BaseConfiguration))
}

func TestCommonProvider_LoadFromConfigFile(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config.Certificates = append(config.Certificates, v)

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 1 != len(config.Certificates) {
		errors.GenerateError(t, fmt.Sprintf("Certificates length %d not corresponding to received %d", 1, len(config.Certificates)))
	}
}

func TestCommonProvider_LoadFromConfigFile2(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
	}

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 0 != len(config.Certificates) {
		errors.GenerateError(t, fmt.Sprintf("Certificates length %d not corresponding to received %d", 0, len(config.Certificates)))
	}
}

func TestCommonProvider_LoadFromConfigFile3(t *testing.T) {
	configChannel := make(chan int)
	config := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
	}

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 0 != len(config.Certificates) {
		errors.GenerateError(t, fmt.Sprintf("Certificates length %d not corresponding to received %d", 0, len(config.Certificates)))
	}
}

func TestCommonProvider_LoadFromConfigFile4(t *testing.T) {
	configChannel := make(chan int)
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	config := &tls.Config{
		Certificates:       []tls.Certificate{v},
	}

	var providers []CommonProvider
	providers = append(providers, CommonProvider{})
	for _, provider := range providers {
		provider.LoadFromConfigFile(config, &configChannel)
	}

	if 1 != len(config.Certificates) {
		errors.GenerateError(t, fmt.Sprintf("Certificates length %d not corresponding to received %d", 1, len(config.Certificates)))
	}
}
