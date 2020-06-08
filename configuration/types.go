package configuration

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
	"os"
)

// Port config
type Port struct {
	Web string `yaml:"web"`
	TLS string `yaml:"tls"`
}

//Cache config
type Cache struct {
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

//Configuration holder
type Configuration struct {
	Redis           Redis    `yaml:"redis"`
	TTL             string   `yaml:"ttl"`
	SSLProviders    []string `yaml:"ssl_providers"`
	ReverseProxyURL string   `yaml:"reverse_proxy_url"`
	Regex           Regex    `yaml:"regex"`
	Cache           Cache    `yaml:"cache"`
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
	data := readFile(os.Getenv("GOPATH") + "/src/github.com/darkweak/souin/configuration/configuration.yml")
	var config Configuration
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
	return config
}
