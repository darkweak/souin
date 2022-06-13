package beego

import (
	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

//Configuration holder
type Configuration struct {
	DefaultCache  *configurationtypes.DefaultCache                       `json:"default_cache" yaml:"default_cache"`
	API           configurationtypes.API                                 `json:"api" yaml:"api"`
	CacheKeys     map[configurationtypes.RegValue]configurationtypes.Key `yaml:"cache_keys"`
	URLs          map[string]configurationtypes.URL                      `json:"urls" yaml:"urls"`
	LogLevel      string                                                 `json:"log_level" yaml:"log_level"`
	Logger        *zap.Logger
	Ykeys         map[string]configurationtypes.SurrogateKeys `json:"ykeys" yaml:"ykeys"`
	SurrogateKeys map[string]configurationtypes.SurrogateKeys `json:"surrogate_keys" yaml:"surrogate_keys"`
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
	return c.Logger
}

// SetLogger set the logger
func (c *Configuration) SetLogger(l *zap.Logger) {
	c.Logger = l
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
	return c.CacheKeys
}

var _ configurationtypes.AbstractConfigurationInterface = (*Configuration)(nil)
