package configuration

// Port config
type Port struct {
	Web string `yaml:"web"`
	TLS string `yaml:"tls"`
}

//Cache config
type Cache struct {
	Headers   []string `yaml:"headers"`
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
	Headers   []string `yaml:"headers"`
}

//DefaultCache configuration
type DefaultCache struct {
	Headers   []string `yaml:"headers"`
	Port      Port     `yaml:"port"`
	Redis     Redis    `yaml:"redis"`
	Regex     Regex    `yaml:"regex"`
	TTL       string   `yaml:"ttl"`
}

// AbstractConfiguration interface
type AbstractConfigurationInterface interface {
	Parse(data []byte) error
	GetUrls() map[string]URL
	GetReverseProxyURL() string
	GetSSLProviders() []string
	GetDefaultCache() DefaultCache
}
