package configurationtypes

import (
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"time"
)

// Duration is the super object to wrap the duration and be able to parse it from the configuration
type Duration struct {
	time.Duration
}

// MarshalYAML transform the Duration into a time.duration object
func (d Duration) MarshalYAML() (interface{}, error) {
	return yaml.Marshal(d.String())
}

// UnmarshalYAML parse the time.duration into a Duration object
func (d Duration) UnmarshalYAML(b *yaml.Node) error {
	var e error
	d.Duration, e = time.ParseDuration(b.Value)

	return e
}

// Port config
type Port struct {
	Web string `json:"web" yaml:"web"`
	TLS string `json:"tls" yaml:"tls"`
}

//Cache config
type Cache struct {
	Headers []string `json:"headers" yaml:"headers"`
	Port    Port     `json:"port" yaml:"port"`
}

//Regex config
type Regex struct {
	Exclude string `json:"exclude" yaml:"exclude"`
}

//URL configuration
type URL struct {
	TTL     Duration `json:"ttl" yaml:"ttl"`
	Headers []string `json:"headers" yaml:"headers"`
}

//CacheProvider config
type CacheProvider struct {
	URL           string      `json:"url" yaml:"url"`
	Path          string      `json:"path" yaml:"path"`
	Configuration interface{} `json:"configuration" yaml:"configuration"`
}

//DefaultCache configuration
type DefaultCache struct {
	Distributed bool          `json:"distributed" yaml:"distributed"`
	Headers     []string      `json:"headers" yaml:"headers"`
	Olric       CacheProvider `json:"olric" yaml:"olric"`
	Port        Port          `json:"port" yaml:"port"`
	Regex       Regex         `json:"regex" yaml:"regex"`
	TTL         Duration      `json:"ttl" yaml:"ttl"`
}

// GetDistributed returns if it uses Olric or not as provider
func (d *DefaultCache) GetDistributed() bool {
	return d.Distributed
}

// GetHeaders returns the default headers that should be cached
func (d *DefaultCache) GetHeaders() []string {
	return d.Headers
}

// GetOlric returns olric configuration
func (d *DefaultCache) GetOlric() CacheProvider {
	return d.Olric
}

// GetRegex returns the regex that shouldn't be cached
func (d *DefaultCache) GetRegex() Regex {
	return d.Regex
}

// GetTTL returns the default TTL
func (d *DefaultCache) GetTTL() time.Duration {
	return d.TTL.Duration
}

// DefaultCacheInterface interface
type DefaultCacheInterface interface {
	GetDistributed() bool
	GetOlric() CacheProvider
	GetHeaders() []string
	GetRegex() Regex
	GetTTL() time.Duration
}

// APIEndpoint is the minimal structure to define an endpoint
type APIEndpoint struct {
	BasePath string `json:"basepath" yaml:"basepath"`
	Enable   bool   `json:"enable" yaml:"enable"`
	Security bool   `json:"security" yaml:"security"`
}

// User is the minimal structure to define a user
type User struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// SecurityAPI object contains informations related to the endpoints
type SecurityAPI struct {
	BasePath string `json:"basepath" yaml:"basepath"`
	Enable   bool   `json:"enable" yaml:"enable"`
	Secret   string `json:"secret" yaml:"secret"`
	Users    []User `json:"users" yaml:"users"`
}

// API structure contains all additional endpoints
type API struct {
	BasePath string      `json:"basepath" yaml:"basepath"`
	Souin    APIEndpoint `json:"souin" yaml:"souin"`
	Security SecurityAPI `json:"security" yaml:"security"`
}

// YKey structure define the way ykeys are stored
type YKey struct {
	URL     string            `json:"url" yaml:"url"`
	Headers map[string]string `json:"headers" yaml:"headers"`
}

// AbstractConfigurationInterface interface
type AbstractConfigurationInterface interface {
	GetUrls() map[string]URL
	GetDefaultCache() DefaultCacheInterface
	GetAPI() API
	GetLogLevel() string
	GetLogger() *zap.Logger
	SetLogger(*zap.Logger)
	GetYkeys() map[string]YKey
}
