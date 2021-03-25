package caddy

import (
	"github.com/darkweak/souin/configurationtypes"
)

//Configuration holder
type Configuration struct {
	DefaultCache    configurationtypes.DefaultCache   `yaml:"default_cache"`
	API             configurationtypes.API            `yaml:"api"`
	URLs            map[string]configurationtypes.URL `yaml:"urls"`
}

// GetUrls get the urls list in the configuration
func (c *Configuration) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetDefaultCache get the default cache
func (c *Configuration) GetDefaultCache() configurationtypes.DefaultCache {
	return c.DefaultCache
}

// GetAPI get the default cache
func (c *Configuration) GetAPI() configurationtypes.API {
	return c.API
}
