package roadrunner

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

var dummyBadgerConfig = []byte(`
api:
  basepath: /httpcache_api
  prometheus:
    basepath: /anything-for-prometheus-metrics
  souin:
    basepath: /httpcache
default_cache:
  allowed_http_verbs:
    - GET
    - POST
    - HEAD
  cdn:
    api_key: XXXX
    dynamic: true
    hostname: XXXX
    network: XXXX
    provider: fastly
    strategy: soft
  headers:
    - Authorization
  regex:
    exclude: '/excluded'
  ttl: 5s
  stale: 10s
log_level: debug
urls:
  'https:\/\/domain.com\/first-.+':
    ttl: 1000s
  'https:\/\/domain.com\/second-route':
    ttl: 10s
    headers:
      - Authorization
  'https?:\/\/mysubdomain\.domain\.com':
    ttl: 50s
    default_cache_control: no-cache
    headers:
      - Authorization
      - 'Content-Type'
ykeys:
  The_First_Test:
    headers:
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
surrogate_keys:
  The_First_Test:
    headers:
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
`)

var dummyConfig = []byte(`
api:
  basepath: /souin-api
  prometheus:
    basepath: /anything-for-prometheus-metrics
  souin:
    basepath: /anything-for-souin
cache_keys:
  .+:
    disable_body: true
    disable_host: true
    disable_method: true
default_cache:
  allowed_http_verbs:
    - GET
    - POST
    - HEAD
  badger:
    url: /badger/url
    path: /badger/path
    configuration:
      SyncEnable: false
  cdn:
    api_key: XXXX
    dynamic: true
    hostname: XXXX
    network: XXXX
    provider: fastly
    strategy: soft
  etcd:
    url: /etcd/url
    path: /etcd/path
    configuration:
      SyncEnable: false
  headers:
    - Authorization
  nuts:
    url: /etcd/url
    path: /etcd/path
    configuration:
      SyncEnable: false
  olric:
    url: 'olric:3320'
    path: /olric/path
    configuration:
      SyncEnable: false
  regex:
    exclude: 'ARegexHere'
  ttl: 10s
  stale: 10s
  default_cache_control: no-store
log_level: debug
urls:
  'https:\/\/domain.com\/first-.+':
    ttl: 1000s
  'https:\/\/domain.com\/second-route':
    ttl: 10s
    headers:
      - Authorization
  'https?:\/\/mysubdomain\.domain\.com':
    ttl: 50s
    default_cache_control: no-cache
    headers:
      - Authorization
      - 'Content-Type'
ykeys:
  The_First_Test:
    headers:
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
surrogate_keys:
  The_First_Test:
    headers:
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
`)

type dummyConfiguration struct{}

func (d *dummyConfiguration) UnmarshalKey(name string, out interface{}) error {
	return nil
}

func (d *dummyConfiguration) Unmarshal(out interface{}) error {
	return nil
}

func (d *dummyConfiguration) Get(name string) interface{} {
	var c map[string]interface{}
	_ = yaml.Unmarshal(dummyConfig, &c)

	return c
}

func (d *dummyConfiguration) Overwrite(values map[string]interface{}) error {
	return nil
}

func (d *dummyConfiguration) Has(name string) bool {
	return true
}

func (d *dummyConfiguration) GracefulTimeout() time.Duration {
	return time.Second
}

func (d *dummyConfiguration) RRVersion() string {
	return "dummy"
}

func Test_ParseConfiguration(t *testing.T) {
	_ = parseConfiguration(&dummyConfiguration{})
}
