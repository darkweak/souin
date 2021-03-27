package caddy

import (
	"github.com/darkweak/souin/configurationtypes"
)

// CaddyDefaultCache the struct
type CaddyDefaultCache struct {
	Headers []string
	Regex   configurationtypes.Regex
	TTL     string
}

// GetHeaders returns the default headers that should be cached
func (d *CaddyDefaultCache) GetHeaders() []string {
	return d.Headers
}

// GetRegex returns the regex that shouldn't be cached
func (d *CaddyDefaultCache) GetRegex() configurationtypes.Regex {
	return d.Regex
}

// GetTTL returns the default TTL
func (d *CaddyDefaultCache) GetTTL() string {
	return d.TTL
}

//Configuration holder
type Configuration struct {
	DefaultCache *CaddyDefaultCache
	API          configurationtypes.API
	URLs         map[string]configurationtypes.URL
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
