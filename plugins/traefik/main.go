package traefik

import (
	"context"
	"github.com/darkweak/souin/cache/coalescing"
	providers2 "github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
)

// Config the Souin configuration.
type Config struct {
	DefaultCache    configurationtypes.DefaultCache   `yaml:"default_cache"`
	ReverseProxyURL string                            `yaml:"reverse_proxy_url"`
	SSLProviders    []string                          `yaml:"ssl_providers"`
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
	rc coalescing.RequestCoalescingInterface,
) {
	responses := make(chan types.ReverseResponse)

	go func() {
		if http.MethodGet == req.Method {
			r, _ := rfc.CachedResponse(
				retriever.GetProvider(),
				req,
				rfc.GetCacheKey(req),
				retriever.GetTransport(),
				true,
			)
			responses <- r
			if nil != r.Response {
				return
			}
		}
	}()

	if http.MethodGet == req.Method {
		response, open := <-responses
		if open && nil != response.Response {
			close(responses)
			for k, v := range response.Response.Header {
				res.Header().Set(k, v[0])
			}
			b, _ := ioutil.ReadAll(response.Response.Body)
			_, _ = res.Write(b)
			return
		}
	}

	close(responses)
	rc.Temporise(req, res, retriever)
}

// SouinTraefikPlugin declaration.
type SouinTraefikPlugin struct {
	next http.Handler
	name string

	Retriever         types.RetrieverResponsePropertiesInterface
	RequestCoalescing coalescing.RequestCoalescingInterface
}

// New create Souin instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	rc := coalescing.Initialize()
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
		RequestCoalescing: rc,
	}, nil
}

func (e *SouinTraefikPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	coalescing.ServeResponse(rw, req, e.Retriever, callback, e.RequestCoalescing)
	e.next.ServeHTTP(rw, req)
}
