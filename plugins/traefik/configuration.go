package souin

import (
	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

//Configuration holder
type Configuration struct {
	DefaultCache *configurationtypes.DefaultCache  `json:"default_cache" yaml:"default_cache"`
	API          configurationtypes.API            `json:"api" yaml:"api"`
	URLs         map[string]configurationtypes.URL `json:"urls" yaml:"urls"`
	LogLevel     string                            `json:"log_level" yaml:"log_level"`
	Logger       *zap.Logger
	Ykeys        map[string]configurationtypes.YKey `json:"ykeys" yaml:"ykeys"`
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
func (c *Configuration) GetYkeys() map[string]configurationtypes.YKey {
	return c.Ykeys
}
