package httpcache

import (
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

// DefaultCache the struct
type DefaultCache struct {
	AllowedHTTPVerbs    []string
	Badger              configurationtypes.CacheProvider
	CDN                 configurationtypes.CDN
	DefaultCacheControl string
	Distributed         bool
	Headers             []string
	Key                 configurationtypes.Key
	Olric               configurationtypes.CacheProvider
	Etcd                configurationtypes.CacheProvider
	Nuts                configurationtypes.CacheProvider
	Regex               configurationtypes.Regex
	TTL                 configurationtypes.Duration
	Stale               configurationtypes.Duration
}

// GetAllowedHTTPVerbs returns the allowed verbs to cache
func (d *DefaultCache) GetAllowedHTTPVerbs() []string {
	return d.AllowedHTTPVerbs
}

// GetBadger returns the Badger configuration
func (d *DefaultCache) GetBadger() configurationtypes.CacheProvider {
	return d.Badger
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

// GetRegex returns the regex that shouldn't be cached
func (d *DefaultCache) GetRegex() configurationtypes.Regex {
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

// GetDefaultCacheControl returns the configured default cache control value
func (d *DefaultCache) GetDefaultCacheControl() string {
	return d.DefaultCacheControl
}

//Configuration holder
type Configuration struct {
	DefaultCache *DefaultCache
	API          configurationtypes.API
	CfgCacheKeys map[string]configurationtypes.Key
	URLs         map[string]configurationtypes.URL
	LogLevel     string
	cacheKeys    map[configurationtypes.RegValue]configurationtypes.Key
	logger       *zap.Logger
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
