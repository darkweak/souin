package configuration

import (
	"io/ioutil"
	"log"
	"gopkg.in/yaml.v3"
)
//Configuration holder
type Configuration struct {
	DefaultCache    DefaultCache   `yaml:"default_cache"`
	ReverseProxyURL string         `yaml:"reverse_proxy_url"`
	SSLProviders    []string       `yaml:"ssl_providers"`
	URLs            map[string]URL `yaml:"urls"`
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

func (c *Configuration) GetUrls() map[string]URL {
	return c.URLs
}

func (c *Configuration) GetReverseProxyURL() string {
	return c.ReverseProxyURL
}

func (c *Configuration) GetSSLProviders() []string {
	return c.SSLProviders
}

func (c *Configuration) GetDefaultCache() DefaultCache {
	return c.DefaultCache
}

// GetConfig allow to retrieve Souin configuration through yaml file
func GetConfiguration() AbstractConfigurationInterface {
	data := readFile("./configuration/configuration.yml")
	var config Configuration
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
	return &config
}
