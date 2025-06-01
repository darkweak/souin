package tests

import (
	"fmt"
	"log"
	"os"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/storages/core"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

// DOMAIN is the domain constant
const DOMAIN = "domain.com"

// PATH is the path constant
const PATH = "/testing"

type testConfiguration struct {
	DefaultCache         *configurationtypes.DefaultCache  `yaml:"default_cache"`
	CacheKeys            configurationtypes.CacheKeys      `yaml:"cache_keys"`
	API                  configurationtypes.API            `yaml:"api"`
	ReverseProxyURL      string                            `yaml:"reverse_proxy_url"`
	SSLProviders         []string                          `yaml:"ssl_providers"`
	URLs                 map[string]configurationtypes.URL `yaml:"urls"`
	LogLevel             string                            `yaml:"log_level"`
	logger               core.Logger
	PluginName           string
	Ykeys                map[string]configurationtypes.SurrogateKeys `yaml:"ykeys"`
	SurrogateKeys        map[string]configurationtypes.SurrogateKeys `yaml:"surrogate_keys"`
	SurrogateKeyDisabled bool                                        `yaml:"disable_surrogate_key"`
}

// GetUrls get the urls list in the configuration
func (c *testConfiguration) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetReverseProxyURL get the reverse proxy url
func (c *testConfiguration) GetReverseProxyURL() string {
	return c.ReverseProxyURL
}

// GetSSLProviders get the ssl providers
func (c *testConfiguration) GetSSLProviders() []string {
	return c.SSLProviders
}

// GetPluginName get the plugin name
func (c *testConfiguration) GetPluginName() string {
	return c.PluginName
}

// GetDefaultCache get the default cache
func (c *testConfiguration) GetDefaultCache() configurationtypes.DefaultCacheInterface {
	return c.DefaultCache
}

// GetAPI get the default cache
func (c *testConfiguration) GetAPI() configurationtypes.API {
	return c.API
}

// GetLogLevel get the log level
func (c *testConfiguration) GetLogLevel() string {
	return c.LogLevel
}

// GetLogger get the logger
func (c *testConfiguration) GetLogger() core.Logger {
	return c.logger
}

// SetLogger set the logger
func (c *testConfiguration) SetLogger(l core.Logger) {
	c.logger = l
}

// GetYkeys get the ykeys list
func (c *testConfiguration) GetYkeys() map[string]configurationtypes.SurrogateKeys {
	return c.Ykeys
}

// GetSurrogateKeys get the surrogate keys list
func (c *testConfiguration) GetSurrogateKeys() map[string]configurationtypes.SurrogateKeys {
	return c.SurrogateKeys
}

// GetSurrogateKeys get the surrogate keys list
func (c *testConfiguration) IsSurrogateDisabled() bool {
	return c.SurrogateKeyDisabled
}

// GetCacheKeys get the cache keys rules to override
func (c *testConfiguration) GetCacheKeys() configurationtypes.CacheKeys {
	return c.CacheKeys
}

var _ configurationtypes.AbstractConfigurationInterface = (*testConfiguration)(nil)

// BaseConfiguration is the legacy configuration
func BaseConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
ykeys:
  The_First_Test:
    headers:
      Authorization: '.+'
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
  The_Third_Test:
`
}

// CDNConfiguration is the CDN configuration
func CDNConfiguration() string {
	return `
api:
  basepath: /souin-api
  souin:
    enable: true
default_cache:
  ttl: 1000s
  cdn:
    dynamic: true
    strategy: hard
`
}

// BadgerConfiguration simulate the configuration for the Badger storage
func BadgerConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  badger:
    configuration:
      syncWrites: true
      readOnly: false
      inMemory: false
      metricsEnabled: true
  distributed: true
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
`
}

// OtterConfiguration simulate the configuration for the Otter storage
func OtterConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  otter: 
    configuration: {}
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
`
}

// NutsConfiguration simulate the configuration for the Nuts storage
func NutsConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  nuts:
    path: "./nuts"
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
`
}

// EtcdConfiguration simulate the configuration for the Etcd storage
func EtcdConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  etcd:
    configuration:
      endpoints:
        - http://etcd:2379
  distributed: true
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
`
}

// RedisConfiguration simulate the configuration for the Redis storage
func RedisConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  redis:
    url: redis:6379
  distributed: true
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
`
}

// SimpleFSConfiguration simulate the configuration for the SimpleFS storage
func SimpleFSConfiguration() string {
	return `
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  simplefs:
    path: "./simplefs"
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
`
}

func baseEmbeddedOlricConfiguration(path string) string {
	return fmt.Sprintf(`
api:
  basepath: /souin-api
  security:
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
  souin:
    enable: true
default_cache:
  distributed: true
  headers:
    - Authorization
  olric:
    %s
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
    headers:
      - Authorization
      - 'Content-Type'
`, path)
}

// OlricConfiguration is the olric included configuration
func OlricConfiguration() string {
	return baseEmbeddedOlricConfiguration(fmt.Sprintf("url: '%s'", "olric:3320"))
}

// EmbeddedOlricPlainConfigurationWithoutAdditionalYAML simulate the configuration for the embedded Olric storage
func EmbeddedOlricPlainConfigurationWithoutAdditionalYAML() string {
	return baseEmbeddedOlricConfiguration(`
    configuration:
      olricd:
        bindAddr: "0.0.0.0"
        bindPort: 3320
        serializer: "msgpack"
        keepAlivePeriod: "20s"
        bootstrapTimeout: "5s"
        partitionCount:  271
        replicaCount: 2
        writeQuorum: 1
        readQuorum: 1
        readRepair: false
        replicationMode: 1 # sync mode. for async, set 1
        tableSize: 1048576 # 1MB in bytes
        memberCountQuorum: 1

      logging:
        verbosity: 6
        level: "DEBUG"
        output: "stderr"
      
      memberlist:
        environment: "local"
        bindAddr: "0.0.0.0"
        bindPort: 3322
        enableCompression: false
        joinRetryInterval: "10s"
        maxJoinAttempts: 2
      
      storageEngines:
        config:
          kvstore:
            tableSize: 4096
`)
}

// EmbeddedOlricConfiguration is the olric included configuration
func EmbeddedOlricConfiguration() string {
	path := "/tmp/olric.yml"
	_ = os.WriteFile(
		path,
		[]byte(
			`
olricd:
  bindAddr: "0.0.0.0"
  bindPort: 3320
  serializer: "msgpack"
  keepAlivePeriod: "300s"
  bootstrapTimeout: "5s"
  partitionCount:  271
  replicaCount: 2
  writeQuorum: 1
  readQuorum: 1
  readRepair: false
  replicationMode: 1 # sync mode. for async, set 1
  tableSize: 1048576 # 1MB in bytes
  memberCountQuorum: 1

logging:
  verbosity: 6
  level: "DEBUG"
  output: "stderr"

memberlist:
  environment: "local"
  bindAddr: "0.0.0.0"
  bindPort: 3322
  enableCompression: false
  joinRetryInterval: "1s"
  maxJoinAttempts: 10

storageEngines:
  config:
    kvstore:
      tableSize: 4096
`),
		0600,
	)

	return baseEmbeddedOlricConfiguration(fmt.Sprintf("path: '%s'", path))
}

// MockConfiguration is an helper to mock the configuration
func MockConfiguration(configurationToLoad func() string) *testConfiguration {
	var config testConfiguration
	if e := yaml.Unmarshal([]byte(configurationToLoad()), &config); e != nil {
		log.Fatal(e)
	}
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	logger, _ := cfg.Build()
	config.SetLogger(logger.Sugar())

	return &config
}
