package configurationtypes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v3"
)

type CacheKey map[RegValue]Key
type CacheKeys []CacheKey

func (c *CacheKeys) parseJSON(rootDecoder *json.Decoder) {
	var token json.Token
	var err error

	_, _ = rootDecoder.Token()
	_, _ = rootDecoder.Token()
	_, _ = rootDecoder.Token()

	for err == nil {
		token, err = rootDecoder.Token()
		key := Key{}
		rg := fmt.Sprint(token)

		value := fmt.Sprint(token)
		if value == "}" || token == nil {
			continue
		}
		for value != "}" && token != nil {
			token, _ = rootDecoder.Token()
			value = fmt.Sprint(token)
			switch value {
			case "disable_body":
				val, _ := rootDecoder.Token()
				key.DisableBody, _ = strconv.ParseBool(fmt.Sprint(val))
			case "disable_host":
				val, _ := rootDecoder.Token()
				key.DisableHost, _ = strconv.ParseBool(fmt.Sprint(val))
			case "disable_method":
				val, _ := rootDecoder.Token()
				key.DisableMethod, _ = strconv.ParseBool(fmt.Sprint(val))
			case "disable_query":
				val, _ := rootDecoder.Token()
				key.DisableQuery, _ = strconv.ParseBool(fmt.Sprint(val))
			case "disable_scheme":
				val, _ := rootDecoder.Token()
				key.DisableScheme, _ = strconv.ParseBool(fmt.Sprint(val))
			case "hash":
				val, _ := rootDecoder.Token()
				key.Hash, _ = strconv.ParseBool(fmt.Sprint(val))
			case "hide":
				val, _ := rootDecoder.Token()
				key.Hide, _ = strconv.ParseBool(fmt.Sprint(val))
			case "template":
				val, _ := rootDecoder.Token()
				key.Template = fmt.Sprint(val)
			case "headers":
				val, _ := rootDecoder.Token()
				key.Headers = []string{}
				for fmt.Sprint(val) != "]" {
					val, _ = rootDecoder.Token()
					header := fmt.Sprint(val)
					if header != "]" {
						key.Headers = append(key.Headers, header)
					}
				}
			}
		}
		*c = append(*c, CacheKey{
			{Regexp: regexp.MustCompile(rg)}: key,
		})
	}
}

func (c *CacheKeys) UnmarshalYAML(value *yaml.Node) error {
	for i := 0; i < len(value.Content)/2; i++ {
		var cacheKey CacheKey
		err := value.Decode(&cacheKey)
		if err != nil {
			return err
		}
		*c = append(*c, cacheKey)
	}

	return nil
}

func (c *CacheKeys) UnmarshalJSON(value []byte) error {
	c.parseJSON(json.NewDecoder(bytes.NewBuffer(value)))

	return nil
}

func (c *CacheKeys) MarshalJSON() ([]byte, error) {
	var strKeys []string
	for _, cacheKey := range *c {
		for rg, key := range cacheKey {
			keyBytes, _ := json.Marshal(key)
			strKeys = append(strKeys, fmt.Sprintf(`"%s": %v`, rg.Regexp.String(), string(keyBytes)))
		}
	}

	return []byte(fmt.Sprintf(`{%s}`, strings.Join(strKeys, ","))), nil
}

// Duration is the super object to wrap the duration and be able to parse it from the configuration
type Duration struct {
	time.Duration
}

// MarshalYAML transform the Duration into a time.duration object
func (d *Duration) MarshalYAML() (interface{}, error) {
	return yaml.Marshal(d.Duration.String())
}

// UnmarshalYAML parse the time.duration into a Duration object
func (d *Duration) UnmarshalYAML(b *yaml.Node) (e error) {
	d.Duration, e = time.ParseDuration(b.Value) // nolint

	return
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
	// Prevent the from being cached matching the regex
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
	// Found to determine if we can use that storage.
	Found bool `json:"found" yaml:"found"`
	// URL to connect to the storage system.
	URL string `json:"url" yaml:"url"`
	// Path to the configuration file.
	Path string `json:"path" yaml:"path"`
	// Declare the cache provider directly in the Souin configuration.
	Configuration interface{} `json:"configuration" yaml:"configuration"`
}

// Timeout configuration to handle the cache provider and the
// reverse-proxy timeout.
type Timeout struct {
	Backend Duration `json:"backend" yaml:"backend"`
	Cache   Duration `json:"cache" yaml:"cache"`
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
	DisableBody   bool     `json:"disable_body,omitempty" yaml:"disable_body,omitempty"`
	DisableHost   bool     `json:"disable_host,omitempty" yaml:"disable_host,omitempty"`
	DisableMethod bool     `json:"disable_method,omitempty" yaml:"disable_method,omitempty"`
	DisableQuery  bool     `json:"disable_query,omitempty" yaml:"disable_query,omitempty"`
	DisableScheme bool     `json:"disable_scheme,omitempty" yaml:"disable_scheme,omitempty"`
	Hash          bool     `json:"hash,omitempty" yaml:"hash,omitempty"`
	Hide          bool     `json:"hide,omitempty" yaml:"hide,omitempty"`
	Template      string   `json:"template,omitempty" yaml:"template,omitempty"`
	Headers       []string `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// DefaultCache configuration
type DefaultCache struct {
	AllowedHTTPVerbs    []string      `json:"allowed_http_verbs" yaml:"allowed_http_verbs"`
	Badger              CacheProvider `json:"badger" yaml:"badger"`
	CDN                 CDN           `json:"cdn" yaml:"cdn"`
	CacheName           string        `json:"cache_name" yaml:"cache_name"`
	Distributed         bool          `json:"distributed" yaml:"distributed"`
	Headers             []string      `json:"headers" yaml:"headers"`
	Key                 Key           `json:"key" yaml:"key"`
	Etcd                CacheProvider `json:"etcd" yaml:"etcd"`
	Mode                string        `json:"mode" yaml:"mode"`
	Nuts                CacheProvider `json:"nuts" yaml:"nuts"`
	Olric               CacheProvider `json:"olric" yaml:"olric"`
	Otter               CacheProvider `json:"otter" yaml:"otter"`
	Redis               CacheProvider `json:"redis" yaml:"redis"`
	Port                Port          `json:"port" yaml:"port"`
	Regex               Regex         `json:"regex" yaml:"regex"`
	Stale               Duration      `json:"stale" yaml:"stale"`
	Storers             []string      `json:"storers" yaml:"storers"`
	Timeout             Timeout       `json:"timeout" yaml:"timeout"`
	TTL                 Duration      `json:"ttl" yaml:"ttl"`
	DefaultCacheControl string        `json:"default_cache_control" yaml:"default_cache_control"`
	MaxBodyBytes        uint64        `json:"max_cacheable_body_bytes" yaml:"max_cacheable_body_bytes"`
	DisableCoalescing   bool          `json:"disable_coalescing" yaml:"disable_coalescing"`
}

// GetAllowedHTTPVerbs returns the allowed verbs to cache
func (d *DefaultCache) GetAllowedHTTPVerbs() []string {
	return d.AllowedHTTPVerbs
}

// GetBadger returns the Badger configuration
func (d *DefaultCache) GetBadger() CacheProvider {
	return d.Badger
}

// GetCacheName returns the cache name to use in the Cache-Status response header
func (d *DefaultCache) GetCacheName() string {
	return d.CacheName
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

// GetMode returns mode configuration
func (d *DefaultCache) GetMode() string {
	return d.Mode
}

// GetNuts returns nuts configuration
func (d *DefaultCache) GetNuts() CacheProvider {
	return d.Nuts
}

// GetOtter returns otter configuration
func (d *DefaultCache) GetOtter() CacheProvider {
	return d.Otter
}

// GetOlric returns olric configuration
func (d *DefaultCache) GetOlric() CacheProvider {
	return d.Olric
}

// GetRedis returns olric configuration
func (d *DefaultCache) GetRedis() CacheProvider {
	return d.Redis
}

// GetRegex returns the regex that shouldn't be cached
func (d *DefaultCache) GetRegex() Regex {
	return d.Regex
}

// GetTimeout returns the backend and cache timeouts
func (d *DefaultCache) GetTimeout() Timeout {
	return d.Timeout
}

// GetTTL returns the default TTL
func (d *DefaultCache) GetTTL() time.Duration {
	return d.TTL.Duration
}

// GetStale returns the stale duration
func (d *DefaultCache) GetStale() time.Duration {
	return d.Stale.Duration
}

// GetStale returns the stale duration
func (d *DefaultCache) GetStorers() []string {
	return d.Storers
}

// GetDefaultCacheControl returns the default Cache-Control response header value when empty
func (d *DefaultCache) GetDefaultCacheControl() string {
	return d.DefaultCacheControl
}

// GetMaxBodyBytes returns the default maximum body size (in bytes) for storing into cache
func (d *DefaultCache) GetMaxBodyBytes() uint64 {
	return d.MaxBodyBytes
}

// IsCoalescingDisable returns if the coalescing is disabled
func (d *DefaultCache) IsCoalescingDisable() bool {
	return d.DisableCoalescing
}

// DefaultCacheInterface interface
type DefaultCacheInterface interface {
	GetAllowedHTTPVerbs() []string
	GetBadger() CacheProvider
	GetCacheName() string
	GetCDN() CDN
	GetDistributed() bool
	GetEtcd() CacheProvider
	GetMode() string
	GetOtter() CacheProvider
	GetNuts() CacheProvider
	GetOlric() CacheProvider
	GetRedis() CacheProvider
	GetHeaders() []string
	GetKey() Key
	GetRegex() Regex
	GetStale() time.Duration
	GetStorers() []string
	GetTimeout() Timeout
	GetTTL() time.Duration
	GetDefaultCacheControl() string
	GetMaxBodyBytes() uint64
	IsCoalescingDisable() bool
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
	Debug      APIEndpoint `json:"debug" yaml:"debug"`
	Prometheus APIEndpoint `json:"prometheus" yaml:"prometheus"`
	Souin      APIEndpoint `json:"souin" yaml:"souin"`
	Security   SecurityAPI `json:"security" yaml:"security"`
}

type SurrogateConfiguration struct {
	Storer string `json:"storer" yaml:"storer"`
}

// SurrogateKeys structure define the way surrogate keys are stored
type SurrogateKeys struct {
	SurrogateConfiguration
	URL     string            `json:"url" yaml:"url"`
	Headers map[string]string `json:"headers" yaml:"headers"`
}

// AbstractConfigurationInterface interface
type AbstractConfigurationInterface interface {
	GetUrls() map[string]URL
	GetPluginName() string
	GetDefaultCache() DefaultCacheInterface
	GetAPI() API
	GetLogLevel() string
	GetLogger() *zap.Logger
	SetLogger(*zap.Logger)
	GetYkeys() map[string]SurrogateKeys
	GetSurrogateKeys() map[string]SurrogateKeys
	GetCacheKeys() CacheKeys
}
