package configuration

import (
	"log"
	"os"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Configuration holder
type Configuration struct {
	DefaultCache    *configurationtypes.DefaultCache                         `yaml:"default_cache"`
	CacheKeys       []map[configurationtypes.RegValue]configurationtypes.Key `yaml:"cache_keys"`
	API             configurationtypes.API                                   `yaml:"api"`
	ReverseProxyURL string                                                   `yaml:"reverse_proxy_url"`
	SSLProviders    []string                                                 `yaml:"ssl_providers"`
	URLs            map[string]configurationtypes.URL                        `yaml:"urls"`
	LogLevel        string                                                   `yaml:"log_level"`
	logger          *zap.Logger
	Ykeys           map[string]configurationtypes.SurrogateKeys `yaml:"ykeys"`
	SurrogateKeys   map[string]configurationtypes.SurrogateKeys `yaml:"surrogate_keys"`
}

func readFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	return data
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

// GetReverseProxyURL get the reverse proxy url
func (c *Configuration) GetReverseProxyURL() string {
	return c.ReverseProxyURL
}

// GetSSLProviders get the ssl providers
func (c *Configuration) GetSSLProviders() []string {
	return c.SSLProviders
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
func (c *Configuration) GetYkeys() map[string]configurationtypes.SurrogateKeys {
	return c.Ykeys
}

// GetSurrogateKeys get the surrogate keys list
func (c *Configuration) GetSurrogateKeys() map[string]configurationtypes.SurrogateKeys {
	return c.SurrogateKeys
}

// GetCacheKeys get the cache keys rules to override
func (c *Configuration) GetCacheKeys() []map[configurationtypes.RegValue]configurationtypes.Key {
	return c.CacheKeys
}

// GetConfiguration allow to retrieve Souin configuration through yaml file
func GetConfiguration() *Configuration {
	data := readFile("./configuration/configuration.yml")
	var config Configuration
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}
	return &config
}

var _ configurationtypes.AbstractConfigurationInterface = (*Configuration)(nil)
