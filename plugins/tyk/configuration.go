package main

import (
	"encoding/json"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

// Duration is the super object to handle durations in the configuration
type Duration struct {
	time.Duration
}

// MarshalJSON will marshall the duration into a string duration
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON will unmarshall the string duration into a time.Duration
func (d *Duration) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		sd := string(b[1 : len(b)-1])
		d.Duration, _ = time.ParseDuration(sd)
		return nil
	}

	var id int64
	id, _ = json.Number(string(b)).Int64()
	d.Duration = time.Duration(id)

	return nil
}

// DefaultCache the struct
type DefaultCache struct {
	Badger      configurationtypes.CacheProvider `json:"badger,omitempty"`
	Distributed bool
	Headers     []string                         `json:"api,omitempty"`
	Olric       configurationtypes.CacheProvider `json:"olric,omitempty"`
	Regex       configurationtypes.Regex         `json:"regex,omitempty"`
	TTL         Duration                         `json:"ttl,omitempty"`
}

// GetBadger returns the Badger configuration
func (d *DefaultCache) GetBadger() configurationtypes.CacheProvider {
	return d.Badger
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
func (d *DefaultCache) GetTTL() time.Duration {
	return d.TTL.Duration
}

//Configuration holder
type Configuration struct {
	DefaultCache *DefaultCache                     `json:"default_cache,omitempty"`
	API          configurationtypes.API            `json:"api,omitempty"`
	URLs         map[string]configurationtypes.URL `json:"urls,omitempty"`
	LogLevel     string                            `json:"log_level,omitempty"`
	logger       *zap.Logger
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
