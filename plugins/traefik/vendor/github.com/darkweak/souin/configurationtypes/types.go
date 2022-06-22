package configurationtypes

import (
	"encoding/json"
	"regexp"
	"time"

	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v3"
)

// Duration is the super object to wrap the duration and be able to parse it from the configuration
type Duration struct {
	time.Duration
}

// MarshalYAML transform the Duration into a time.duration object
func (d *Duration) MarshalYAML() (interface{}, error) {
	return yaml.Marshal(d.Duration.String())
}

// UnmarshalYAML parse the time.duration into a Duration object
func (d *Duration) UnmarshalYAML(b *yaml.Node) error {
	var e error
	d.Duration, e = time.ParseDuration(b.Value) // nolint

	return e
}

// MarshalJSON transform the Duration into a time.duration object
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

// UnmarshalJSON parse the time.duration into a Duration object
func (d *Duration) UnmarshalJSON(b []byte) error {
	sd := string(b[1 : len(b)-1])
	d.Duration, _ = time.ParseDuration(sd)
	return nil
}

// Port config
type Port struct {
	Web string `json:"web" yaml:"web"`
	TLS string `json:"tls" yaml:"tls"`
}

// Cache config
type Cache struct {
	Headers []string `json:"headers" yaml:"headers"`
	Port    Port     `json:"port" yaml:"port"`
}

// Regex config
type Regex struct {
	Exclude string `json:"exclude" yaml:"exclude"`
}

// RegValue represent a valid regexp as value
type RegValue struct {
	*regexp.Regexp
}

func (r *RegValue) UnmarshalYAML(b *yaml.Node) error {
	r.Regexp = regexp.MustCompile(b.Value)

	return nil
}

// UnmarshalJSON parse the string configuration into a compiled regexp.
func (r *RegValue) UnmarshalJSON(b []byte) error {
	r.Regexp = regexp.MustCompile(string(b))

	return nil
}

// URL configuration
type URL struct {
	TTL                 Duration `json:"ttl" yaml:"ttl"`
	Headers             []string `json:"headers" yaml:"headers"`
	DefaultCacheControl string   `json:"default_cache_control" yaml:"default_cache_control"`
}

// CacheProvider config
type CacheProvider struct {
	// URL to connect to the storage system.
	URL string `json:"url" yaml:"url"`
	// Path to the configuration file.
	Path string `json:"path" yaml:"path"`
	// Declare the cache provider directly in the Souin configuration.
	Configuration interface{} `json:"configuration" yaml:"configuration"`
}

// CDN config
type CDN struct {
	APIKey    string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Dynamic   bool   `json:"dynamic,omitempty" yaml:"dynamic,omitempty"`
	Email     string `json:"email,omitempty" yaml:"email,omitempty"`
	Hostname  string `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	Network   string `json:"network,omitempty" yaml:"network,omitempty"`
	Provider  string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Strategy  string `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	ServiceID string `json:"service_id,omitempty" yaml:"service_id,omitempty"`
	ZoneID    string `json:"zone_id,omitempty" yaml:"zone_id,omitempty"`
}

type Key struct {
	DisableBody   bool `json:"disable_body" yaml:"disable_body"`
	DisableHost   bool `json:"disable_host" yaml:"disable_host"`
	DisableMethod bool `json:"disable_method" yaml:"disable_method"`
}

// DefaultCache configuration
type DefaultCache struct {
	AllowedHTTPVerbs    []string      `json:"allowed_http_verbs" yaml:"allowed_http_verbs"`
	Badger              CacheProvider `json:"badger" yaml:"badger"`
	CDN                 CDN           `json:"cdn" yaml:"cdn"`
	Distributed         bool          `json:"distributed" yaml:"distributed"`
	Headers             []string      `json:"headers" yaml:"headers"`
	Key                 Key           `json:"key" yaml:"key"`
	Etcd                CacheProvider `json:"etcd" yaml:"etcd"`
	Nuts                CacheProvider `json:"nuts" yaml:"nuts"`
	Olric               CacheProvider `json:"olric" yaml:"olric"`
	Port                Port          `json:"port" yaml:"port"`
	Regex               Regex         `json:"regex" yaml:"regex"`
	TTL                 Duration      `json:"ttl" yaml:"ttl"`
	Stale               Duration      `json:"stale" yaml:"stale"`
	DefaultCacheControl string        `json:"default_cache_control" yaml:"default_cache_control"`
}

// GetAllowedHTTPVerbs returns the allowed verbs to cache
func (d *DefaultCache) GetAllowedHTTPVerbs() []string {
	return d.AllowedHTTPVerbs
}

// GetBadger returns the Badger configuration
func (d *DefaultCache) GetBadger() CacheProvider {
	return d.Badger
}

// GetCDN returns the CDN configuration
func (d *DefaultCache) GetCDN() CDN {
	return d.CDN
}

// GetDistributed returns if it uses Olric or not as provider
func (d *DefaultCache) GetDistributed() bool {
	return d.Distributed
}

// GetHeaders returns the default headers that should be cached
func (d *DefaultCache) GetHeaders() []string {
	return d.Headers
}

// GetKey returns the default Key generation strategy
func (d *DefaultCache) GetKey() Key {
	return d.Key
}

// GetEtcd returns etcd configuration
func (d *DefaultCache) GetEtcd() CacheProvider {
	return d.Etcd
}

// GetNuts returns nuts configuration
func (d *DefaultCache) GetNuts() CacheProvider {
	return d.Nuts
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

// GetStale returns the stale duration
func (d *DefaultCache) GetStale() time.Duration {
	return d.Stale.Duration
}

// GetDefaultCacheControl returns the default Cache-Control response header value when empty
func (d *DefaultCache) GetDefaultCacheControl() string {
	return d.DefaultCacheControl
}

// DefaultCacheInterface interface
type DefaultCacheInterface interface {
	GetAllowedHTTPVerbs() []string
	GetBadger() CacheProvider
	GetCDN() CDN
	GetDistributed() bool
	GetEtcd() CacheProvider
	GetNuts() CacheProvider
	GetOlric() CacheProvider
	GetHeaders() []string
	GetKey() Key
	GetRegex() Regex
	GetTTL() time.Duration
	GetStale() time.Duration
	GetDefaultCacheControl() string
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
	BasePath   string      `json:"basepath" yaml:"basepath"`
	Prometheus APIEndpoint `json:"prometheus" yaml:"prometheus"`
	Souin      APIEndpoint `json:"souin" yaml:"souin"`
	Security   SecurityAPI `json:"security" yaml:"security"`
}

// SurrogateKeys structure define the way surrogate keys are stored
type SurrogateKeys struct {
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
	GetYkeys() map[string]SurrogateKeys
	GetSurrogateKeys() map[string]SurrogateKeys
	GetCacheKeys() map[RegValue]Key
}
