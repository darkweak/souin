package caddy

import (
	"github.com/darkweak/souin/configurationtypes"
)

//Configuration holder
type Configuration struct {
	DefaultCache    configurationtypes.DefaultCache   `yaml:"default_cache"`
	API             configurationtypes.API            `yaml:"api"`
	ReverseProxyURL string                            `yaml:"reverse_proxy_url"`
	SSLProviders    []string                          `yaml:"ssl_providers"`
	URLs            map[string]configurationtypes.URL `yaml:"urls"`
}

// Parse configuration
func (c *Configuration) Parse(_ []byte) error {
	return nil
}

// GetUrls get the urls list in the configuration
func (c *Configuration) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetReverseProxyURL get the reverse proxy url
func (c *Configuration) GetReverseProxyURL() string {
	return c.ReverseProxyURL
}

// GetSSLProviders get the ssl providers
func (c *Configuration) GetSSLProviders() []string {
	return c.SSLProviders
}

// GetDefaultCache get the default cache
func (c *Configuration) GetDefaultCache() configurationtypes.DefaultCache {
	return c.DefaultCache
}

// GetAPI get the default cache
func (c *Configuration) GetAPI() configurationtypes.API {
	return c.API
}
