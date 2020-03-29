package configuration

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Port config
type Port struct {
	Web string `yaml:"web"`
	Tls string `yaml:"tls"`
}

//Cache config
type Cache struct {
	Mode string `yaml:"mode"`
	Port Port   `yaml:"port"`
}

//Redis config
type Redis struct {
	Url string `yaml:"url"`
}

//Regex config
type Regex struct {
	Exclude string `yaml:"exclude"`
}

//Configuration holder
type Configuration struct {
	Redis           Redis  `yaml:"redis"`
	TTL             string `yaml:"ttl"`
	ReverseProxyUrl string `yaml:"reverse_proxy_url"`
	Regex           Regex  `yaml:"regex"`
	Cache           Cache  `yaml:"cache"`
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
	configFile := "./configuration.yml"
	data := readFile(configFile)
	var config Configuration
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
	return config
}
