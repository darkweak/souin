package main

import (
	"github.com/darkweak/souin/configurationtypes"
)

// DefaultCache the struct
type DefaultCache struct {
	Distributed bool
	Headers     []string                         `json:"api,omitempty"`
	Olric       configurationtypes.CacheProvider `json:"olric,omitempty"`
	Regex       configurationtypes.Regex         `json:"regex,omitempty"`
	TTL         string                           `json:"ttl,omitempty"`
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
	DefaultCache *DefaultCache                     `json:"default_cache,omitempty"`
	API          configurationtypes.API            `json:"api,omitempty"`
	URLs         map[string]configurationtypes.URL `json:"urls,omitempty"`
	LogLevel     string                            `json:"log_level,omitempty"`
	logger       interface{}
	Ykeys        map[string]configurationtypes.YKey `json:"ykeys,omitempty"`
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
func (c *Configuration) GetLogger() interface{} {
	return c.logger
}

// SetLogger set the logger
func (c *Configuration) SetLogger(l interface{}) {
	c.logger = l
}

// GetYkeys get the ykeys list
func (c *Configuration) GetYkeys() map[string]configurationtypes.YKey {
	return c.Ykeys
}
