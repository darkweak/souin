package middleware

import (
	"github.com/darkweak/souin/configurationtypes"
)

// BaseConfiguration holder
type BaseConfiguration struct {
	DefaultCache  *configurationtypes.DefaultCache  `json:"default_cache" yaml:"default_cache"`
	API           configurationtypes.API            `json:"api" yaml:"api"`
	CacheKeys     configurationtypes.CacheKeys      `json:"cache_keys" yaml:"cache_keys"`
	URLs          map[string]configurationtypes.URL `json:"urls" yaml:"urls"`
	LogLevel      string                            `json:"log_level" yaml:"log_level"`
	PluginName    string
	Ykeys         map[string]configurationtypes.SurrogateKeys `json:"ykeys" yaml:"ykeys"`
	SurrogateKeys map[string]configurationtypes.SurrogateKeys `json:"surrogate_keys" yaml:"surrogate_keys"`
}

// GetUrls get the urls list in the configuration
func (c *BaseConfiguration) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetPluginName get the plugin name
func (c *BaseConfiguration) GetPluginName() string {
	return c.PluginName
}

// GetDefaultCache get the default cache
func (c *BaseConfiguration) GetDefaultCache() configurationtypes.DefaultCacheInterface {
	return c.DefaultCache
}

// GetAPI get the default cache
func (c *BaseConfiguration) GetAPI() configurationtypes.API {
	return c.API
}

// GetLogLevel get the log level
func (c *BaseConfiguration) GetLogLevel() string {
	return c.LogLevel
}

// GetYkeys get the ykeys list
func (c *BaseConfiguration) GetYkeys() map[string]configurationtypes.SurrogateKeys {
	return c.SurrogateKeys
}

// GetSurrogateKeys get the surrogate keys list
func (c *BaseConfiguration) GetSurrogateKeys() map[string]configurationtypes.SurrogateKeys {
	return c.SurrogateKeys
}

// GetCacheKeys get the cache keys rules to override
func (c *BaseConfiguration) GetCacheKeys() configurationtypes.CacheKeys {
	return c.CacheKeys
}

var _ configurationtypes.AbstractConfigurationInterface = (*BaseConfiguration)(nil)
