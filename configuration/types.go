package configuration

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Port config
type Port struct {
	Web string `yaml:"web"`
	TLS string `yaml:"tls"`
}

//Cache config
type Cache struct {
	Headers   []string `yaml:"headers"`
	Providers []string `yaml:"providers"`
	Port      Port     `yaml:"port"`
}

//Redis config
type Redis struct {
	URL string `yaml:"url"`
}

//Regex config
type Regex struct {
	Exclude string `yaml:"exclude"`
}

//URL configuration
type URL struct {
	TTL       string   `yaml:"ttl"`
	Providers []string `yaml:"providers"`
	Headers   []string `yaml:"headers"`
}

//DefaultCache configuration
type DefaultCache struct {
	Headers   []string `yaml:"headers"`
	Port      Port     `yaml:"port"`
	Providers []string `yaml:"providers"`
	Redis     Redis    `yaml:"redis"`
	Regex     Regex    `yaml:"regex"`
	TTL       string   `yaml:"ttl"`
}

//Configuration holder
type Configuration struct {
	DefaultCache    DefaultCache   `yaml:"default_cache"`
	ReverseProxyURL string         `yaml:"reverse_proxy_url"`
	SSLProviders    []string       `yaml:"ssl_providers"`
	URLs            map[string]URL `yaml:"urls"`
}

// Parse configuration
func (c *Configuration) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}
	return nil
}

func readFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

// GetConfig allow to retrieve Souin configuration through yaml file
func GetConfig() Configuration {
	data := readFile("/configuration.yml")
	var config Configuration
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
	return config
}
