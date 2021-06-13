package caddy

import (
	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

// DefaultCache the struct
type DefaultCache struct {
	Distributed bool
	Headers     []string
	Olric       configurationtypes.CacheProvider
	Regex       configurationtypes.Regex
	TTL         string
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
func (d *DefaultCache) GetOlric() configurationtypes.CacheProvider {
	return d.Olric
}

// GetRegex returns the regex that shouldn't be cached
func (d *DefaultCache) GetRegex() configurationtypes.Regex {
	return d.Regex
}

// GetTTL returns the default TTL
func (d *DefaultCache) GetTTL() string {
	return d.TTL
}

//Configuration holder
type Configuration struct {
	DefaultCache *DefaultCache
	API          configurationtypes.API
	URLs         map[string]configurationtypes.URL
	LogLevel     string
	logger       *zap.Logger
	Ykeys        map[string]configurationtypes.YKey
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
func (c *Configuration) GetYkeys() map[string]configurationtypes.YKey {
	return c.Ykeys
}
