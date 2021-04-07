package traefik

import (
	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

//Configuration holder
type Configuration struct {
	DefaultCache *configurationtypes.DefaultCache  `yaml:"default_cache"`
	API          configurationtypes.API            `yaml:"api"`
	URLs         map[string]configurationtypes.URL `yaml:"urls"`
	LogLevel     string                            `yaml:"log_level"`
	logger       *zap.Logger
}

// Parse configuration
func (c *Configuration) Parse(data []byte) error {
	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}
	return nil
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
