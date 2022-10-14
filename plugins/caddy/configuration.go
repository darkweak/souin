package httpcache

import (
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

// DefaultCache the struct
type DefaultCache struct {
	// Allowed HTTP verbs to be cached by the system.
	AllowedHTTPVerbs []string `json:"allowed_http_verbs"`
	// Badger provider configuration.
	Badger configurationtypes.CacheProvider `json:"badger"`
	// The cache name to use in the Cache-Status response header.
	CacheName string                 `json:"cache_name"`
	CDN       configurationtypes.CDN `json:"cdn"`
	// The default Cache-Control header value if none set by the upstream server.
	DefaultCacheControl string `json:"default_cache_control"`
	// Redis provider configuration.
	Distributed bool `json:"distributed"`
	// Headers to add to the cache key if they are present.
	Headers []string `json:"headers"`
	// Configure the global key generation.
	Key configurationtypes.Key `json:"key"`
	// Olric provider configuration.
	Olric configurationtypes.CacheProvider `json:"olric"`
	// Redis provider configuration.
	Redis configurationtypes.CacheProvider `json:"redis"`
	// Etcd provider configuration.
	Etcd configurationtypes.CacheProvider `json:"etcd"`
	// NutsDB provider configuration.
	Nuts configurationtypes.CacheProvider `json:"nuts"`
	// Regex to exclude cache.
	Regex configurationtypes.Regex `json:"regex"`
	// Time before cache or backend access timeout.
	Timeout configurationtypes.Timeout `json:"timeout"`
	// Time to live.
	TTL configurationtypes.Duration `json:"ttl"`
	// Stale time to live.
	Stale configurationtypes.Duration `json:"stale"`
}

// GetAllowedHTTPVerbs returns the allowed verbs to cache
func (d *DefaultCache) GetAllowedHTTPVerbs() []string {
	return d.AllowedHTTPVerbs
}

// GetBadger returns the Badger configuration
func (d *DefaultCache) GetBadger() configurationtypes.CacheProvider {
	return d.Badger
}

// GetCacheName returns the cache name to use in the Cache-Status response header
func (d *DefaultCache) GetCacheName() string {
	return d.CacheName
}

// GetCDN returns the CDN configuration
func (d *DefaultCache) GetCDN() configurationtypes.CDN {
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
func (d *DefaultCache) GetKey() configurationtypes.Key {
	return d.Key
}

// GetEtcd returns etcd configuration
func (d *DefaultCache) GetEtcd() configurationtypes.CacheProvider {
	return d.Etcd
}

// GetNuts returns nuts configuration
func (d *DefaultCache) GetNuts() configurationtypes.CacheProvider {
	return d.Nuts
}

// GetOlric returns olric configuration
func (d *DefaultCache) GetOlric() configurationtypes.CacheProvider {
	return d.Olric
}

// GetRedis returns redis configuration
func (d *DefaultCache) GetRedis() configurationtypes.CacheProvider {
	return d.Redis
}

// GetRegex returns the regex that shouldn't be cached
func (d *DefaultCache) GetRegex() configurationtypes.Regex {
	return d.Regex
}

// GetTimeout returns the backend and cache timeouts
func (d *DefaultCache) GetTimeout() configurationtypes.Timeout {
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

// GetDefaultCacheControl returns the configured default cache control value
func (d *DefaultCache) GetDefaultCacheControl() string {
	return d.DefaultCacheControl
}

//Configuration holder
type Configuration struct {
	// Default cache to fallback on when none are redefined.
	DefaultCache *DefaultCache
	// API endpoints enablers.
	API configurationtypes.API
	// Cache keys configuration.
	CfgCacheKeys map[string]configurationtypes.Key
	// Override the ttl depending the cases.
	URLs map[string]configurationtypes.URL
	// Logger level, fallback on caddy's one when not redefined.
	LogLevel  string
	cacheKeys map[configurationtypes.RegValue]configurationtypes.Key
	logger    *zap.Logger
}

// GetUrls get the urls list in the configuration
func (c *Configuration) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetDefaultCache get the default cache
func (c *Configuration) GetDefaultCache() configurationtypes.DefaultCacheInterface {
	return c.DefaultCache
}

// GetAPI get the default cache
func (c *Configuration) GetAPI() configurationtypes.API {
	return c.API
}

// GetLogLevel get the log level
func (c *Configuration) GetLogLevel() string {
	return c.LogLevel
}

// GetLogger get the logger
func (c *Configuration) GetLogger() *zap.Logger {
	return c.logger
}

// SetLogger set the logger
func (c *Configuration) SetLogger(l *zap.Logger) {
	c.logger = l
}

// GetYkeys get the ykeys list
func (c *Configuration) GetYkeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}

// GetSurrogateKeys get the surrogate keys list
func (c *Configuration) GetSurrogateKeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}

// GetCacheKeys get the cache keys rules to override
func (c *Configuration) GetCacheKeys() map[configurationtypes.RegValue]configurationtypes.Key {
	return c.cacheKeys
}

var _ configurationtypes.AbstractConfigurationInterface = (*Configuration)(nil)
