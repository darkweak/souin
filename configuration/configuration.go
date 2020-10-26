package configuration

import (
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v3"
	"github.com/darkweak/souin/configuration_types"
)

//Configuration holder
type Configuration struct {
	DefaultCache    configuration_types.DefaultCache   `yaml:"default_cache"`
	ReverseProxyURL string                             `yaml:"reverse_proxy_url"`
	SSLProviders    []string                           `yaml:"ssl_providers"`
	URLs            map[string]configuration_types.URL `yaml:"urls"`
}

func readFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

// Parse configuration
func (c *Configuration) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}
	return nil
}

// GetUrls get the urls list in the configuration
func (c *Configuration) GetUrls() map[string]configuration_types.URL {
	return c.URLs
}

// GetReverseProxyURL get the reverse proxy url
func (c *Configuration) GetReverseProxyURL() string {
	return c.ReverseProxyURL
}

// GetSSLProviders get the ssl providers
func (c *Configuration) GetSSLProviders() []string {
	return c.SSLProviders
}

// GetDefaultCache get the default cache
func (c *Configuration) GetDefaultCache() configuration_types.DefaultCache {
	return c.DefaultCache
}

// GetConfiguration allow to retrieve Souin configuration through yaml file
func GetConfiguration() configuration_types.AbstractConfigurationInterface {
	data := readFile("./configuration/configuration.yml")
	var config Configuration
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
	return &config
}
