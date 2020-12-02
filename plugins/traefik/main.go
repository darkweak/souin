package traefik

import (
	"context"
	"encoding/json"
	"fmt"
	providers2 "github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"gopkg.in/yaml.v3"
	"net/http"
)

// Config the Souin configuration.
type Config struct {
	DefaultCache    configurationtypes.DefaultCache   `yaml:"default_cache"`
	ReverseProxyURL string                             `yaml:"reverse_proxy_url"`
	SSLProviders    []string                           `yaml:"ssl_providers"`
	URLs            map[string]configurationtypes.URL `yaml:"urls"`
}

// Parse configuration
func (c *Config) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}
	return nil
}

// GetUrls get the urls list in the configuration
func (c *Config) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetReverseProxyURL get the reverse proxy url
func (c *Config) GetReverseProxyURL() string {
	return c.ReverseProxyURL
}

// GetSSLProviders get the ssl providers
func (c *Config) GetSSLProviders() []string {
	return c.SSLProviders
}

// GetDefaultCache get the default cache
func (c *Config) GetDefaultCache() configurationtypes.DefaultCache {
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
				if 0 < len(r.Response) {
					alreadyHaveResponse = true
				}
			}
		}
	}()

	if http.MethodGet == req.Method {
		response, open := <-responses
		if open && http.MethodGet == req.Method && 0 < len(response.Response) {
			close(responses)
			var responseJSON types.RequestResponse
			err := json.Unmarshal(response.Response, &responseJSON)
			if err != nil {
				fmt.Println(err)
			}
			for k, v := range responseJSON.Headers {
				res.Header().Set(k, v[0])
			}
			_, _ = res.Write(responseJSON.Body)
		}
	}
}

// SouinTraefikPlugin declaration.
type SouinTraefikPlugin struct {
	next http.Handler
	name string

	Retriever types.RetrieverResponsePropertiesInterface
}

// New create Souin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	provider := providers2.InitializeProvider(config)
	regexpUrls := helpers.InitializeRegexp(config)

	return &SouinTraefikPlugin{
		name: name,
		next: next,
		Retriever: &types.RetrieverResponseProperties{
			MatchedURL: configurationtypes.URL{
				TTL:     config.GetDefaultCache().TTL,
				Headers: config.GetDefaultCache().Headers,
			},
			Provider:      provider,
			Configuration: config,
			RegexpUrls:    regexpUrls,
		},
	}, nil
}

func (e *SouinTraefikPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	service.ServeResponse(rw, req, e.Retriever, callback)
	e.next.ServeHTTP(rw, req)
}
