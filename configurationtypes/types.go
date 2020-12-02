package configurationtypes

// Port config
type Port struct {
	Web string `yaml:"web"`
	TLS string `yaml:"tls"`
}

//Cache config
type Cache struct {
	Headers []string `yaml:"headers"`
	Port    Port     `yaml:"port"`
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
	TTL     string   `yaml:"ttl"`
	Providers []string `yaml:"cache_providers"`
	Headers []string `yaml:"headers"`
}

//DefaultCache configuration
type DefaultCache struct {
	Headers []string `yaml:"headers"`
	Port    Port     `yaml:"port"`
	Providers []string `yaml:"cache_providers"`
	Redis   Redis    `yaml:"redis"`
	Regex   Regex    `yaml:"regex"`
	TTL     string   `yaml:"ttl"`
}

// APIEndpoint is the minimal structure to define an endpoint
type APIEndpoint struct {
	BasePath string `yaml:"basepath"`
	Enable   bool   `yaml:"enable"`
	Security bool   `yaml:"security"`
}

// User is the minimal structure to define a user
type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// SecurityAPI object contains informations related to the endpoints
type SecurityAPI struct {
	BasePath string `yaml:"basepath"`
	Enable   bool   `yaml:"enable"`
	Secret   string `yaml:"secret"`
	Users    []User `yaml:"users"`
}

// API structure contains all additional endpoints
type API struct {
	BasePath string      `yaml:basepath`
	Souin    APIEndpoint `yaml:"souin"`
	Security SecurityAPI `yaml:"security"`
}

// AbstractConfigurationInterface interface
type AbstractConfigurationInterface interface {
	Parse(data []byte) error
	GetUrls() map[string]URL
	GetReverseProxyURL() string
	GetSSLProviders() []string
	GetDefaultCache() DefaultCache
	GetAPI() API
}
