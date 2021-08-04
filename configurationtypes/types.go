package configurationtypes

import (
	"go.uber.org/zap"
	"time"
)

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

//Regex config
type Regex struct {
	Exclude string `yaml:"exclude"`
}

//URL configuration
type URL struct {
	TTL     time.Duration `yaml:"ttl"`
	Headers []string      `yaml:"headers"`
}

//CacheProvider config
type CacheProvider struct {
	URL           string      `yaml:"url" json:"url"`
	Path          string      `yaml:"path" json:"path"`
	Configuration interface{} `yaml:"configuration" json:"configuration"`
}

//DefaultCache configuration
type DefaultCache struct {
	Distributed bool          `yaml:"distributed"`
	Headers     []string      `yaml:"headers"`
	Olric       CacheProvider `yaml:"olric"`
	Port        Port          `yaml:"port"`
	Regex       Regex         `yaml:"regex"`
	TTL         time.Duration `yaml:"ttl"`
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
	return d.TTL
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

type YKey struct {
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers"`
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
