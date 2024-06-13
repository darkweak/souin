package tests

import (
	"fmt"
	"log"
	"os"

	"github.com/darkweak/souin/plugins/souin/configuration"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DOMAIN is the domain constant
const DOMAIN = "domain.com"

// PATH is the path constant
const PATH = "/testing"

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

// EtcdConfiguration simulate the configuration for the Nuts storage
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

// RedisConfiguration simulate the configuration for the Nuts storage
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
    url: localhost:6379
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
func MockConfiguration(configurationToLoad func() string) *configuration.Configuration {
	var config configuration.Configuration
	e := config.Parse([]byte(configurationToLoad()))
	if e != nil {
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
	config.SetLogger(logger)

	return &config
}
