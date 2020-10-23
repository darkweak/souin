package traefik

import (
	"context"
	"net/http"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/service"
	"crypto/tls"
	"github.com/darkweak/souin/providers"
	providers2 "github.com/darkweak/souin/cache/providers"
	"encoding/json"
	"fmt"
)

// Config the Souin configuration.
type Config struct {
	DefaultCache    configuration.DefaultCache   `yaml:"default_cache"`
	ReverseProxyURL string                       `yaml:"reverse_proxy_url"`
	SSLProviders    []string                     `yaml:"ssl_providers"`
	URLs            map[string]configuration.URL `yaml:"urls"`
}

// Parse configuration
func (c *Config) Parse(data []byte) error {
	return nil
}

func (c *Config) GetUrls() map[string]configuration.URL {
	return c.URLs
}

func (c *Config) GetReverseProxyURL() string {
	return c.ReverseProxyURL
}

func (c *Config) GetSSLProviders() []string {
	return c.SSLProviders
}

func (c *Config) GetDefaultCache() configuration.DefaultCache {
	return c.DefaultCache
}

// CreateConfig creates the Souin plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

func callback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	key string,
) {
	responses := make(chan types.ReverseResponse)

	alreadyHaveResponse := false

	go func() {
		if http.MethodGet == req.Method {
			if !alreadyHaveResponse {
				r := retriever.GetProvider().GetRequestInCache(key)
				responses <- retriever.GetProvider().GetRequestInCache(key)
				if "" != r.Response {
					alreadyHaveResponse = true
				}
			}
		}
	}()

	if http.MethodGet == req.Method {
		response, open := <-responses
		if open && http.MethodGet == req.Method && "" != response.Response {
			close(responses)
			var responseJSON types.RequestResponse
			err := json.Unmarshal([]byte(response.Response), &responseJSON)
			if err != nil {
				fmt.Println(err)
			}
			for k, v := range responseJSON.Headers {
				res.Header().Set(k, v[0])
			}
			res.Write(responseJSON.Body)
		}
	}
}

// SouinTraefikPlugin declaration.
type SouinTraefikPlugin struct {
	next http.Handler
	name string

	Retriever types.RetrieverResponsePropertiesInterface
}

// Create Souin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	configChannel := make(chan int)
	tlsConfig := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		NameToCertificate:  make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("./default/server.crt", "./default/server.key")
	tlsConfig.Certificates = append(tlsConfig.Certificates, v)

	go func() {
		providers.InitProviders(tlsConfig, &configChannel, config)
	}()

	provider := providers2.InitializeProvider(config)
	regexpUrls := helpers.InitializeRegexp(config)

	return &SouinTraefikPlugin{
		name: name,
		next: next,
		Retriever: &types.RetrieverResponseProperties{
			MatchedURL: configuration.URL{
				TTL:       config.GetDefaultCache().TTL,
				Headers:   config.GetDefaultCache().Headers,
			},
			Provider: provider,
			Configuration: config,
			RegexpUrls: regexpUrls,
		},
	}, nil
}

func (e *SouinTraefikPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	service.ServeResponse(rw, req, e.Retriever, callback)
	e.next.ServeHTTP(rw, req)
}
