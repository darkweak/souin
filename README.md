![Souin logo](https://github.com/darkweak/souin/blob/master/docs/img/logo.svg)

# Souin Table of Contents
1. [Souin reverse-proxy cache description](#project-description)
2. [Configuration](#configuration)  
  2.1. [Required configuration](#required-configuration)  
    2.1.1. [Souin as plugin](#souin-as-plugin)  
    2.1.2. [Souin out-of-the-box](#souin-out-of-the-box)  
  2.2. [Optional configuration](#optional-configuration)
3. [Storages](#storages)  
4. [APIs](#apis)  
  4.1. [Prometheus API](#prometheus-api)  
  4.2. [Souin API](#souin-api)  
  4.3. [Security API](#security-api)
5. [Diagrams](#diagrams)  
  5.1. [Sequence diagram](#sequence-diagram)
6. [Cache systems](#cache-systems)
7. [GraphQL](#graphql)  
8. [Plugins](#plugins)  
  8.1. [Beego filter](#beego-filter)  
  8.2. [Caddy module](#caddy-module)  
  8.3. [Chi middleware](#chi-middleware)  
  8.4. [Dotweb middleware](#dotweb-middleware)  
  8.5. [Echo middleware](#echo-middleware)  
  8.6. [Fiber middleware](#fiber-middleware)  
  8.7. [Gin middleware](#gin-middleware)  
  8.8. [Goa middleware](#goa-middleware)  
  8.9. [Go-zero middleware](#go-zero-middleware)  
  8.10. [Goyave middleware](#goyave-middleware)  
  8.11. [Hertz middleware](#hertz-middleware)  
  8.12. [Kratos filter](#kratos-filter)  
  8.13. [Roadrunner middleware](#roadrunner-middleware)  
  8.14. [Skipper filter](#skipper-filter)  
  8.15. [Træfik plugin](#træfik-plugin)  
  8.16. [Tyk plugin](#tyk-plugin)  
  8.17. [Webgo middleware](#webgo-middleware)  
9. [Credits](#credits)

# Souin HTTP cache

## Project description
Souin is a new HTTP cache system suitable for every reverse-proxy. It can be either placed on top of your current reverse-proxy whether it's Apache, Nginx or as plugin in your favorite reverse-proxy like Træfik, Caddy or Tyk.  
Since it's written in go, it can be deployed on any server and thanks to the docker integration, it will be easy to install on top of a Swarm, or a kubernetes instance.  
It's RFC compatible, supporting Vary, request coalescing, stale cache-control and other specifications related to the [RFC-7234](https://tools.ietf.org/html/rfc7234).  
It supports the newly written RFCs (currently in draft) [http-cache-groups](https://datatracker.ietf.org/doc/draft-nottingham-http-cache-groups/) and [http-invalidation](https://datatracker.ietf.org/doc/draft-nottingham-http-invalidation/).  
It also supports the [Cache-Status HTTP response header](https://www.rfc-editor.org/rfc/rfc9211), the YKey group such as Varnish, the [Targeted HTTP Cache Control RFC](https://www.rfc-editor.org/rfc/rfc9213), .  
It supports the ESI tags, thanks to the [go-esi package](https://github.com/darkweak/go-esi).

> [!WARNING]
> Since `v1.7.0` Souin implements only one storage. If you need a specific storage you have to take it from [the storages repository](https://github.com/darkweak/storages) and add it either in your code, during the build otherwise.  
(e.g. with otter using caddy) You have to build your caddy module with the desired storage `xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/otter/caddy` and configure otter in your Caddyfile/JSON configuration file.  
See the [storages section](#storages) or the [documentation website about the storages](https://docs.souin.io/docs/storages).

## Configuration
The configuration file is store at `/anywhere/configuration.yml`. You can supply your own as long as you use one of the minimal configurations below.

### Required configuration
#### Souin as plugin
```yaml
default_cache: # Required
  ttl: 10s # Default TTL
```

#### Souin out-of-the-box
```yaml
default_cache: # Required
  ttl: 10s # Default TTL
reverse_proxy_url: 'http://traefik' # If it's in the same network you can use http://your-service, otherwise just use https://yourdomain.com
```

|  Key                |  Description                                                |  Value example                                                                                                            |
|:--------------------|:------------------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------|
| `default_cache.ttl` | Duration to cache request (in seconds)                      | 10                                                                                                                        |

Besides, it's highly recommended to set `default_cache.default_cache_control` (see it below) to avoid undesired caching for responses without `Cache-Control` header.

### Optional configuration
```yaml
# /anywhere/configuration.yml
api:
  basepath: /souin-api # Default route basepath for every additional APIs to avoid conflicts with existing routes
  prometheus: # Prometheus exposed metrics
    basepath: /anything-for-prometheus-metrics # Change the prometheus endpoint basepath
  souin: # Souin listing keys and cache management
    basepath: /anything-for-souin # Change the souin endpoint basepath
cache_keys:
  '.*\.css':
    disable_body: true # Prevent the body from being used in the cache key
    disable_host: true # Prevent the host from being used in the cache key
    disable_method: true # Prevent the method from being used in the cache key
    disable_query: true # Prevent the query string from being used in the cache key
    disable_scheme: true # request scheme the query string from being used in the cache key
    hash: true # Hash the cache key instead of a plaintext one
    hide: true # Prevent the cache key to be in the response Cache-Status header
    headers: # Add headers to the key
      - Authorization # Add the header value in the key
      - Content-Type # Add the header value in the key
    template: "{http.request.method}-{http.request.host}-{http.request.path}" # Use caddy placeholders to create the key (when this option is enabled, disable_* directives are skipped)
cdn: # If Souin is set after a CDN fill these informations
  api_key: XXXX # Your provider API key if mandatory
  provider: fastly # The provider placed before Souin (e.g. fastly, cloudflare, akamai, varnish)
  strategy: soft # The strategy to purge the CDN cache based on tags (e.g. soft, hard)
  dynamic: true # If true, you'll be able to add custom keys than the ones defined under the surrogate_keys key
default_cache:
  allowed_http_verbs: # Allowed HTTP verbs to cache (default GET, HEAD).
    - GET
    - POST
    - HEAD
  cache_name: Souin # Override the cache name to use in the Cache-Status header
  distributed: true # Use Olric or Etcd distributed storage
  key:
    disable_body: true # Prevent the body from being used in the cache key
    disable_host: true # Prevent the host from being used in the cache key
    disable_method: true # Prevent the method from being used in the cache key
    disable_query: true # Prevent the query string from being used in the cache key
    disable_scheme: true # Prevent the request scheme string from being used in the cache key
    hash: true # Hash the cache key instead of a plaintext one
    hide: true # Prevent the cache key to be in the response Cache-Status header
    headers: # Add headers to the key
      - Authorization # Add the header value in the key
      - Content-Type # Add the header value in the key
    template: "{http.request.method}-{http.request.host}-{http.request.path}" # Use caddy placeholders to create the key (when this option is enabled, disable_* directives are skipped)
  etcd: # If distributed is set to true, you'll have to define either the etcd or olric section
    configuration: # Configure directly the Etcd client
      endpoints: # Define multiple endpoints
        - etcd-1:2379 # First node
        - etcd-2:2379 # Second node
        - etcd-3:2379 # Third node
  mode: bypass # Override the RFC respect.
  olric: # If distributed is set to true, you'll have to define either the etcd or olric section
    url: 'olric:3320' # Olric server
  regex:
    exclude: 'ARegexHere' # Regex to exclude from cache
  stale: 1000s # Stale duration
  timeout: # Timeout configuration
    backend: 10s # Backend timeout before returning an HTTP unavailable response
    cache: 20ms # Cache provider (badger, etcd, nutsdb, olric, depending the configuration you set) timeout before returning a miss
  ttl: 1000s # Default TTL
  default_cache_control: no-store # Set default value for Cache-Control response header if not set by upstream
log_level: INFO # Logs verbosity [ DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL ], case do not matter
ssl_providers: # The {providers}.json to use
  - traefik
urls:
  'https:\/\/domain.com\/first-.+': # First regex route configuration
    ttl: 1000s # Override default TTL
  'https:\/\/domain.com\/second-route': # Second regex route configuration
    ttl: 10s # Override default TTL
  'https?:\/\/mysubdomain\.domain\.com': # Third regex route configuration
    ttl: 50s # Override default TTL'
    default_cache_control: public, max-age=86400 # Override default default Cache-Control
ykeys:
  The_First_Test:
    headers:
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
  The_Third_Test:
  The_Fourth_Test:
surrogate_keys:
  The_First_Test:
    headers:
      Content-Type: '.+'
  The_Second_Test:
    url: 'the/second/.+'
  The_Third_Test:
  The_Fourth_Test:
```

| Key                                               | Description                                                                                                                                 | Value example                                                                                                                                                                                                                 |
|:--------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `api`                                             | The cache-handler API cache management                                                                                                      |                                                                                                                                                                                                                               |
| `api.basepath`                                    | BasePath for all APIs to avoid conflicts                                                                                                    | `/your-non-conflicting-route`<br/><br/>`(default: /souin-api)`                                                                                                                                                                |
| `api.{api}.enable`                                | (DEPRECATED) Enable the API with related routes                                                                                             | `true`<br/><br/>`(default: true if you define the api name, false then)`                                                                                                                                                      |
| `api.{api}.security`                              | (DEPRECATED) Enable the JWT Authentication token verification                                                                               | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `api.security.secret`                             | (DEPRECATED) JWT secret key                                                                                                                 | `Any_charCanW0rk123`                                                                                                                                                                                                          |
| `api.security.users`                              | (DEPRECATED) Array of authorized users with username x password combo                                                                       | `- username: admin`<br/><br/>`  password: admin`                                                                                                                                                                              |
| `api.souin.security`                              | Enable JWT validation to access the resource                                                                                                | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cache_keys`                                      | Define the key generation rules for each URI matching the key regexp                                                                        |                                                                                                                                                                                                                               |
| `cache_keys.{your regexp}`                        | Regexp that the URI should match to override the key generation                                                                             | `.+\.css`                                                                                                                                                                                                                     |
| `cache_keys.{your regexp}.disable_body`           | Disable the body part in the key matching the regexp (GraphQL context)                                                                      | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cache_keys.{your regexp}.disable_host`           | Disable the host part in the key matching the regexp                                                                                        | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cache_keys.{your regexp}.disable_method`         | Disable the method part in the key matching the regexp                                                                                      | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cache_keys.{your regexp}.disable_query`          | Disable the query string part in the key matching the regexp                                                                                | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cache_keys.{your regexp}.disable_scheme`         | Disable the request scheme string part in the key matching the regexp                                                                       | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cache_keys.{your regexp}.hash`                   | Hash the key matching the regexp                                                                                                            | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cache_keys.{your regexp}.headers`                | Add headers to the key matching the regexp                                                                                                  | `- Authorization`<br/><br/>`- Content-Type`<br/><br/>`- X-Additional-Header`                                                                                                                                                  |
| `cache_keys.{your regexp}.hide`                   | Prevent the key from being exposed in the `Cache-Status` HTTP response header                                                               | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `cdn`                                             | The CDN management, if you use any cdn to proxy your requests Souin will handle that                                                        |                                                                                                                                                                                                                               |
| `cdn.provider`                                    | The provider placed before Souin                                                                                                            | `akamai`<br/><br/>`fastly`<br/><br/>`souin`                                                                                                                                                                                   |
| `cdn.api_key`                                     | The api key used to access to the provider                                                                                                  | `XXXX`                                                                                                                                                                                                                        |
| `cdn.dynamic`                                     | Enable the dynamic keys returned by your backend application                                                                                | `false`<br/><br/>`(default: true)`                                                                                                                                                                                            |
| `cdn.email`                                       | The api key used to access to the provider if required, depending the provider                                                              | `XXXX`                                                                                                                                                                                                                        |
| `cdn.hostname`                                    | The hostname if required, depending the provider                                                                                            | `domain.com`                                                                                                                                                                                                                  |
| `cdn.network`                                     | The network if required, depending the provider                                                                                             | `your_network`                                                                                                                                                                                                                |
| `cdn.strategy`                                    | The strategy to use to purge the cdn cache, soft will keep the content as a stale resource                                                  | `hard`<br/><br/>`(default: soft)`                                                                                                                                                                                             |
| `cdn.service_id`                                  | The service id if required, depending the provider                                                                                          | `123456_id`                                                                                                                                                                                                                   |
| `cdn.zone_id`                                     | The zone id if required, depending the provider                                                                                             | `anywhere_zone`                                                                                                                                                                                                               |
| `default_cache.allowed_http_verbs`                | The HTTP verbs to support cache                                                                                                             | `- GET`<br/><br/>`- POST`<br/><br/>`(default: GET, HEAD)`                                                                                                                                                                     |
| `default_cache.badger`                            | Configure the Badger cache storage                                                                                                          |                                                                                                                                                                                                                               |
| `default_cache.badger.path`                       | Configure Badger with a file                                                                                                                | `/anywhere/badger_configuration.json`                                                                                                                                                                                         |
| `default_cache.badger.configuration`              | Configure Badger directly in the Caddyfile or your JSON caddy configuration                                                                 | [See the Badger configuration for the options](https://dgraph.io/docs/badger/get-started/)                                                                                                                                    |
| `default_cache.default_cache_control`             | Set the default value of `Cache-Control` response header if not set by upstream (Souin treats empty `Cache-Control` as `public` if omitted) | `no-store`                                                                                                                                                                                                                    |
| `default_cache.etcd`                              | Configure the Etcd cache storage                                                                                                            |                                                                                                                                                                                                                               |
| `default_cache.etcd.configuration`                | Configure Etcd directly in the Caddyfile or your JSON caddy configuration                                                                   | [See the Etcd configuration for the options](https://pkg.go.dev/go.etcd.io/etcd/clientv3#Config)                                                                                                                              |
| `default_cache.etcd.url`                          | Set the Etcd cluster endpoint                                                                                                               | `http://etcd1:2379,http://etcd2:2379`                                                                                                                                                                                         |
| `default_cache.key`                               | Override the key generation with the ability to disable unecessary parts                                                                    |                                                                                                                                                                                                                               |
| `default_cache.key.disable_body`                  | Disable the body part in the key (GraphQL context)                                                                                          | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `default_cache.key.disable_host`                  | Disable the host part in the key                                                                                                            | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `default_cache.key.disable_method`                | Disable the method part in the key                                                                                                          | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `default_cache.key.disable_query`                 | Disable the query string part in the key                                                                                                    | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `default_cache.key.disable_scheme`                | Disable the request scheme string part in the key                                                                                           | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `default_cache.key.hash`                          | Hash the key name in the storage                                                                                                            | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `default_cache.key.headers`                       | Add headers to the key matching the regexp                                                                                                  | `- Authorization`<br/><br/>`- Content-Type`<br/><br/>`- X-Additional-Header`                                                                                                                                                  |
| `default_cache.key.hide`                          | Prevent the key from being exposed in the `Cache-Status` HTTP response header                                                               | `true`<br/><br/>`(default: false)`                                                                                                                                                                                            |
| `default_cache.key.template`                      | Use caddy placeholders to create the key (when this option is enabled, disable_* directives are skipped)                                    | [Placeholders documentation](https://caddyserver.com/docs/caddyfile/concepts#placeholders)                                                                                                                                    |
| `default_cache.max_cacheable_body_bytes`          | Set the maximum size (in bytes) for a response body to be cached (unlimited if omited)                                                      | `1048576` (1MB)                                                                                                                                                                                                               |
| `default_cache.mode`                              | RFC respect tweaking                                                                                                                        | One of `bypass` `bypass_request` `bypass_response` `strict` (default `strict`)                                                                                                                                                |
| `default_cache.nats`                              | Configure the Nats cache storage                                                                                                            |                                                                                                                                                                                                                               |
| `default_cache.nats.url`                          | Set the Nats cluster endpoint                                                                                                               | `nats://127.0.0.1:4222,nats://127.0.0.1:4223`                                                                                                                                                                                 |
| `default_cache.nats.configuration`                | Configure Nats directly in the Caddyfile or your JSON caddy configuration                                                                   | [See the Nats configuration for the options](https://github.com/nats-io/nats.go/blob/main/nats.go#L267)                                                                                                                       |
| `default_cache.nuts`                              | Configure the Nuts cache storage                                                                                                            |                                                                                                                                                                                                                               |
| `default_cache.nuts.path`                         | Set the Nuts file path storage                                                                                                              | `/anywhere/nuts/storage`                                                                                                                                                                                                      |
| `default_cache.nuts.configuration`                | Configure Nuts directly in the Caddyfile or your JSON caddy configuration                                                                   | [See the Nuts configuration for the options](https://github.com/nutsdb/nutsdb#default-options)                                                                                                                                |
| `default_cache.olric`                             | Configure the Olric cache storage                                                                                                           |                                                                                                                                                                                                                               |
| `default_cache.olric.path`                        | Configure Olric with a file                                                                                                                 | `/anywhere/olric_configuration.json`                                                                                                                                                                                          |
| `default_cache.olric.configuration`               | Configure Olric directly in the Caddyfile or your JSON caddy configuration                                                                  | [See the Olric configuration for the options](https://github.com/buraksezer/olric/blob/master/cmd/olricd/olricd.yaml/)                                                                                                        |
| `default_cache.otter`                             | Configure the Otter cache storage                                                                                                           |                                                                                                                                                                                                                               |
| `default_cache.otter.configuration`               | Configure Otter directly in the Caddyfile or your JSON caddy configuration                                                                  |                                                                                                                                                                                                                               |
| `default_cache.otter.configuration.size`          | Set the size of the pool in Otter                                                                                                           | `999999` (default `10000`)                                                                                                                                                                                                    |
| `default_cache.port.{web,tls}`                    | The device's local HTTP/TLS port that Souin should be listening on                                                                          | Respectively `80` and `443`                                                                                                                                                                                                   |
| `default_cache.redis`                             | Configure the Redis cache storage                                                                                                           |                                                                                                                                                                                                                               |
| `default_cache.redis.url`                         | Set the Redis cluster endpoint                                                                                                              | `nats://127.0.0.1:4222,nats://127.0.0.1:4223`                                                                                                                                                                                 |
| `default_cache.redis.configuration`               | Configure Redis directly in the Caddyfile or your JSON caddy configuration                                                                  | [See the Go-redis configuration for the options](https://github.com/redis/go-redis/blob/master/options.go#L31) or [See the Rueidis configuration for the options](https://github.com/redis/rueidis/blob/master/rueidis.go#56) |
| `default_cache.regex.exclude`                     | The regex used to prevent paths being cached                                                                                                | `^[A-z]+.*$`                                                                                                                                                                                                                  |
| `default_cache.stale`                             | The stale duration                                                                                                                          | `25m`                                                                                                                                                                                                                         |
| `default_cache.timeout`                           | The timeout configuration                                                                                                                   |                                                                                                                                                                                                                               |
| `default_cache.timeout.backend`                   | The timeout duration to consider the backend as unreachable                                                                                 | `10s`                                                                                                                                                                                                                         |
| `default_cache.timeout.cache`                     | The timeout duration to consider the cache provider as unreachable                                                                          | `10ms`                                                                                                                                                                                                                        |
| `default_cache.ttl`                               | The TTL duration                                                                                                                            | `120s`                                                                                                                                                                                                                        |
| `log_level`                                       | The log level                                                                                                                               | `One of DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL it's case insensitive`                                                                                                                                                 |
| `reverse_proxy_url`                               | The reverse-proxy's instance URL (Apache, Nginx, Træfik...)                                                                                 | - `http://yourservice` (Container way)<br/>`http://localhost:81` (Local way)<br/>`http://yourdomain.com:81` (Network way)                                                                                                     |
| `ssl_providers`                                   | List of your providers handling certificates                                                                                                | `- traefik`<br/><br/>`- nginx`<br/><br/>`- apache`                                                                                                                                                                            |
| `urls.{your url or regex}`                        | List of your custom configuration depending each URL or regex                                                                               | 'https:\/\/yourdomain.com'                                                                                                                                                                                                    |
| `urls.{your url or regex}.ttl`                    | Override the default TTL if defined                                                                                                         | `90s`<br/><br/>`10m`                                                                                                                                                                                                          |
| `urls.{your url or regex}.default_cache_control`  | Override the default default `Cache-Control` if defined                                                                                     | `public, max-age=86400`                                                                                                                                                                                                       |
| `surrogate_keys.{key name}.headers`               | Headers that should match to be part of the surrogate key group                                                                             | `Authorization: ey.+`<br/><br/>`Content-Type: json`                                                                                                                                                                           |
| `surrogate_keys.{key name}.headers.{header name}` | Header name that should be present a match the regex to be part of the surrogate key group                                                  | `Content-Type: json`                                                                                                                                                                                                          |
| `surrogate_keys.{key name}.url`                   | Url that should match to be part of the surrogate key group                                                                                 | `.+`                                                                                                                                                                                                                          |
| `ykeys.{key name}.headers`                        | (DEPRECATED) Headers that should match to be part of the ykey group                                                                         | `Authorization: ey.+`<br/><br/>`Content-Type: json`                                                                                                                                                                           |
| `ykeys.{key name}.headers.{header name}`          | (DEPRECATED) Header name that should be present a match the regex to be part of the ykey group                                              | `Content-Type: json`                                                                                                                                                                                                          |
| `ykeys.{key name}.url`                            | (DEPRECATED) Url that should match to be part of the ykey group                                                                             | `.+`                                                                                                                                                                                                                          |

## APIs
All endpoints are accessible through the `api.basepath` configuration line or by default through `/souin-api` to avoid named route conflicts. Be sure to define an unused route to not break your existing application.

### Prometheus API
Prometheus API expose some metrics about the cache.  
The base path for the prometheus API is `/metrics`.
**Not supported inside Træfik because the deny the unsafe library usage inside plugins**

| Method  | Endpoint | Description                             |
|:--------|:---------|:----------------------------------------|
| `GET`   | `/`      | Expose the different keys listed below. |

| Key                                | Definition                                          |
|:-----------------------------------|:----------------------------------------------------|
| `souin_request_upstream_counter`   | Count the incoming requests that go to the upstream |
| `souin_no_cached_response_counter` | Count the uncacheable responses                     |
| `souin_cached_response_counter`    | Count the cacheable responses                       |
| `souin_avg_response_time`          | Average response time                               |

### Souin API
Souin API allow users to manage the cache.  
The base path for the souin API is `/souin`.  
The Souin API supports the invalidation by surrogate keys such as Fastly which will replace the Varnish system. You can read the doc [about this system](https://github.com/darkweak/souin/blob/master/pkg/surrogate/README.md).
This system is able to invalidate by tags your cloud provider cache. Actually it supports Akamai and Fastly but in a near future some other providers would be implemented like Cloudflare or Varnish.

| Method  | Endpoint          | Headers                                                    | Description                                                                                                                                                                         |
|:--------|:------------------|:-----------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `GET`   | `/`               | -                                                          | List stored keys cache                                                                                                                                                              |
| `GET`   | `/surrogate_keys` | -                                                          | List stored keys cache                                                                                                                                                              |
| `PURGE` | `/{id or regexp}` | -                                                          | Purge selected item(s) depending. The parameter can be either a specific key or a regexp                                                                                            |
| `PURGE` | `/?ykey={key}`    | -                                                          | Purge selected item(s) corresponding to the target ykey such as Varnish (deprecated)                                                                                                |
| `PURGE` | `/`               | `Surrogate-Key: Surrogate-Key-First, Surrogate-Key-Second` | Purge selected item(s) belong to the target key in the header `Surrogate-Key` (see [Surrogate-Key system](https://github.com/darkweak/souin/blob/master/cache/surrogate/README.md)) |
| `PURGE` | `/flush`          | -                                                          | Purge all providers and surrogate storages                                                                                                                                          |

### Security API
**DEPRECATED**  
Security API allows users to protect other APIs with JWT authentication.  
The base path for the security API is `/authentication`.

| Method | Endpoint   | Body                                       | Headers                                                                         | Description                                                                                                            |
|:-------|:-----------|:-------------------------------------------|:--------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------|
| `POST` | `/login`   | `{"username":"admin", "password":"admin"}` | `['Content-Type' => 'json']`                                                    | Try to login, it returns a response which contains the cookie name `souin-authorization-token` with the JWT if succeed |
| `POST` | `/refresh` | `-`                                        | `['Content-Type' => 'json', 'Cookie' => 'souin-authorization-token=the-token']` | Refreshes the token, replaces the old with a new one                                                                   |

## Diagrams

### Sequence diagram
See the sequence diagram for the minimal version below
![Sequence diagram](https://github.com/darkweak/souin/blob/master/docs/plantUML/sequenceDiagram.svg?sanitize=true)

## Cache systems
Supported providers
 - [Badger](https://github.com/dgraph-io/badger)
 - [NutsDB](https://github.com/nutsdb/nutsdb)
 - [Etcd](https://github.com/etcd-io/etcd)
 - [Olric](https://github.com/buraksezer/olric)

The cache system sits on top of three providers at the moment. It provides two in-memory storage solutions (badger and nuts), and two distributed storages Olric and Etcd because setting, getting, updating and deleting keys in these providers is as easy as it gets.  
**The Badger provider (default one)**: you can tune its configuration using the badger configuration inside your Souin configuration. In order to do that, you have to declare the `badger` block. See the following json example.
```json
"badger": {
  "configuration": {
    "ValueDir": "default",
    "ValueLogFileSize": 16777216,
    "MemTableSize": 4194304,
    "ValueThreshold": 524288,
    "BypassLockGuard": true
  }
}
```

**The Nuts provider**: you can tune its configuration using the nuts configuration inside your Souin configuration. In order to do that, you have to declare the `nuts` block. See the following json example.
```json
"nuts": {
  "configuration": {
    "Dir": "default",
    "EntryIdxMode": 1,
    "RWMode": 0,
    "SegmentSize": 1024,
    "NodeNum": 42,
    "SyncEnable": true,
    "StartFileLoadingMode": 1
  }
}
```

**The Otter provider**: you can tune its configuration using the otter configuration inside your Souin configuration. In order to do that, you have to declare the `otter` block. See the following json example.
```json
"otter": {
  "configuration": {
    "size": 9999999
  }
}
```

**The Olric provider**: you can tune its configuration using the olric configuration inside your Souin configuration and declare Souin has to use the distributed provider. In order to do that, you have to declare the `olric` block and the `distributed` directive. See the following json example.
```json
"distributed": true,
"olric": {
  "configuration": {
    # Olric configuration here...
  }
}
```
In order to do that, the Olric provider need to be either on the same network as the Souin instance when using docker-compose or over the internet, then it will use by default in-memory to avoid network latency as much as possible. 

**The Etcd provider**: you can tune its configuration using the etcd configuration inside your Souin configuration and declare Souin has to use the distributed provider. In order to do that, you have to declare the `etcd` block and the `distributed` directive. See the following json example.
```json
"distributed": true,
"etcd": {
  "configuration": {
    # Etcd configuration here...
  }
}
```
In order to do that, the Etcd provider need to be either on the same network as the Souin instance when using docker-compose or over the internet, then it will use by default in-memory to avoid network latency as much as possible. 
Souin will return at first the response from the choosen provider when it gives a non-empty response, or fallback to the reverse proxy otherwise.
Since v1.4.2, Souin supports [Olric](https://github.com/buraksezer/olric) and since v1.6.10 it supports [Etcd](https://github.com/etcd-io/etcd) to handle distributed cache.

## GraphQL
This feature is currently in beta.  
Souin can partially cache your GraphQL requests. It automatically handles the data retrieval and omit the caching for the mutations.  
However, it will invalidate whole cache keys with a body when you send a mutation request due to the inability to read and understand automatically which cached endpoint should be deleted.  
You can enable the GraphQL support with the `default_cache.allowed_http_verbs` key to define the list of supported HTTP verbs like `GET`, `POST`, `DELETE`.
```yaml
default_cache:
  allowed_http_verbs:
    - GET
    - POST
    - HEAD
```

### Cache invalidation
The cache invalidation is built for CRUD requests, if you're doing a GET HTTP request, it will serve the cached response when it exists, otherwise the reverse-proxy response will be served.  
If you're doing a POST, PUT, PATCH or DELETE HTTP request, the related cache GET request, and the list endpoint will be dropped.  
It also supports invalidation via [Souin API](#souin-api) to invalidate the cache programmatically.


## Plugins

### Beego filter
To use Souin as beego filter, you can refer to the [Beego filter integration folder](https://github.com/darkweak/souin/tree/master/plugins/beego) to discover how to configure it.  
You just have to define a new beego router and tell to the instance to use the `Handle` method like below:
```go
import (
	"net/http"

	httpcache "github.com/darkweak/souin/plugins/beego"
)

func main(){

    // ...
	web.InsertFilterChain("/*", httpcache.NewHTTPCacheFilter())
    // ...

}
```

### Caddy module
To use Souin as caddy module, you can refer to the [Caddy module integration folder](https://github.com/darkweak/souin/tree/master/plugins/caddy) to discover how to configure it.  
The related Caddyfile can be found [here](https://github.com/darkweak/souin/tree/master/plugins/caddy/Caddyfile).  
Then you just have to run the following command:
```bash
xcaddy build --with github.com/darkweak/souin/plugins/caddy
```

There is the fully configuration below
```caddy
{
    log {
        level debug
    }
    cache {
        allowed_http_verbs GET POST PATCH
        api {
            basepath /some-basepath
            prometheus {
                security
            }
            souin {
                security
            }
        }
        badger {
            path the_path_to_a_file.json
        }
        cache_name Souin
        cache_keys {
            .*\.something {
                disable_body
                disable_host
                disable_method
                disable_query
                disable_scheme
                headers X-Token Authorization
                hide
                hash
            }
        }
        cdn {
            api_key XXXX
            dynamic
            email darkweak@protonmail.com
            hostname domain.com
            network your_network
            provider fastly
            strategy soft
            service_id 123456_id
            zone_id anywhere_zone
        }
        key {
            disable_body
            disable_host
            disable_method
            disable_query
            disable_scheme
            hash
            hide
            headers Content-Type Authorization
        }
        log_level debug
        etcd {
            configuration {
                # Your Etcd configuration here
            }
        }
        olric {
            url url_to_your_cluster:3320
            path the_path_to_a_file.yaml
            configuration {
                # Your Olric configuration here
            }
        }
        regex {
            exclude /test2.*
        }
        stale 200s
        timeout {
          backend 20s
          cache 5ms
        }
        ttl 1000s
        default_cache_control no-store
    }
}

:4443
respond "Hello World!"

@match path /test1*
@match2 path /test2*
@matchdefault path /default
@souin-api path /souin-api*

cache @match {
    ttl 5s
    badger {
        path /tmp/badger/first-match
        configuration {
            # Required value
            ValueDir <string>

            # Optional
            SyncWrites <bool>
            NumVersionsToKeep <int>
            ReadOnly <bool>
            Compression <int>
            InMemory <bool>
            MetricsEnabled <bool>
            MemTableSize <int>
            BaseTableSize <int>
            BaseLevelSize <int>
            LevelSizeMultiplier <int>
            TableSizeMultiplier <int>
            MaxLevels <int>
            VLogPercentile <float>
            ValueThreshold <int>
            NumMemtables <int>
            BlockSize <int>
            BloomFalsePositive <float>
            BlockCacheSize <int>
            IndexCacheSize <int>
            NumLevelZeroTables <int>
            NumLevelZeroTablesStall <int>
            ValueLogFileSize <int>
            ValueLogMaxEntries <int>
            NumCompactors <int>
            CompactL0OnClose <bool>
            LmaxCompaction <bool>
            ZSTDCompressionLevel <int>
            VerifyValueChecksum <bool>
            EncryptionKey <string>
            EncryptionKey <Duration>
            BypassLockGuard <bool>
            ChecksumVerificationMode <int>
            DetectConflicts <bool>
            NamespaceOffset <int>
        }
    }
}

cache @match2 {
    ttl 50s
    badger {
        path /tmp/badger/second-match
        configuration {
            ValueDir match2
            ValueLogFileSize 16777216
            MemTableSize 4194304
            ValueThreshold 524288
            BypassLockGuard true
        }
    }
    default_cache_control "public, max-age=86400"
}

cache @matchdefault {
    ttl 5s
    badger {
        path /tmp/badger/default-match
        configuration {
            ValueDir default
            ValueLogFileSize 16777216
            MemTableSize 4194304
            ValueThreshold 524288
            BypassLockGuard true
        }
    }
}

route /no-method-and-domain.css {
    cache {
        cache_keys {
            .*\.css {
                disable_host
                disable_method
            }
        }
    }
    respond "Hello without storing method and domain cache key"
}

cache @souin-api {}
```

### Chi middleware
To use Souin as chi middleware, you can refer to the [Chi middleware integration folder](https://github.com/darkweak/souin/tree/master/plugins/chi) to discover how to configure it.  
You just have to define a new chi router and tell to the instance to use the `Handle` method like below:
```go
import (
	"net/http"

	cache "github.com/darkweak/souin/plugins/chi"
	"github.com/go-chi/chi/v5"
)

func main(){

    // ...
	router := chi.NewRouter()
	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	router.Use(httpcache.Handle)
	router.Get("/*", defaultHandler)
    // ...

}
```

### Dotweb middleware
To use Souin as dotweb middleware, you can refer to the [Dotweb plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/dotweb) to discover how to configure it.  
You just have to define a new dotweb router and tell to the instance to use the process method like below:
```go
import (
	cache "github.com/darkweak/souin/plugins/dotweb"
	"github.com/go-dotweb/dotweb/v5"
)

func main(){

    // ...
	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	app.HttpServer.GET("/:p", func(ctx dotweb.Context) error {
		return ctx.WriteString("Hello, World 👋!")
	}).Use(httpcache)
    // ...

}
```

### Echo middleware
To use Souin as echo middleware, you can refer to the [Echo plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/echo) to discover how to configure it.  
You just have to define a new echo router and tell to the instance to use the process method like below:
```go
import (
	"net/http"

	souin_echo "github.com/darkweak/souin/plugins/echo"
	"github.com/labstack/echo/v4"
)

func main(){

    // ...
	e := echo.New()
	s := souin_echo.New(souin_echo.DefaultConfiguration)
	e.Use(s.Process)
    // ...

}
```

### Fiber middleware
To use Souin as fiber middleware, you can refer to the [Fiber plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/fiber) to discover how to configure it.  
You just have to define a new fiber router and tell to the instance to use the process method like below:
```go
import (
	cache "github.com/darkweak/souin/plugins/fiber"
	"github.com/gofiber/fiber/v2"
)

func main(){

    // ...
	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	app.Use(httpcache.Handle)
    // ...

}
```

### Gin middleware
To use Souin as gin middleware, you can refer to the [Gin plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/gin) to discover how to configure it.  
You just have to define a new gin router and tell to the instance to use the process method like below:
```go
import (
	"net/http"

	souin_gin "github.com/darkweak/souin/plugins/gin"
	"github.com/gin-gonic/gin"
)

func main(){

    // ...
	r := gin.New()
	s := souin_gin.New(souin_gin.DefaultConfiguration)
	r.Use(s.Process())
    // ...

}
```

### Go-zero middleware
To use Souin as go-zero middleware, you can refer to the [Go-zero plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/go-zero) to discover how to configure it.  
You just have to give a Condfiguration object to the `NewHTTPCache` method to get a new HTTP cache instance and use the Handle method as a GlobalMiddleware:
```go
import (
	"net/http"

	cache "github.com/darkweak/souin/plugins/go-zero"
)

func main(){

    // ...
	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	server.Use(httpcache.Handle)
    // ...

}
```

### Goa middleware
To use Souin as goa middleware, you can refer to the [Goa plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/goa) to discover how to configure it.  
You just have to start Goa, define a new goa router and tell to the router instance to use the Handle method as GlobalMiddleware like below:
```go
import (
	"net/http"

	httpcache "github.com/darkweak/souin/plugins/goa"
	goahttp "goa.design/goa/v3/http"
)

func main(){

    // ...
	g := goahttp.NewMuxer()
	g.Use(httpcache.NewHTTPCache(httpcache.DevDefaultConfiguration))
    // ...

}
```

### Goyave middleware
To use Souin as goyave middleware, you can refer to the [Goyave plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/goyave) to discover how to configure it.  
You just have to start Goyave, define a new goyave router and tell to the router instance to use the Handle method as GlobalMiddleware like below:
```go
import (
	"net/http"

	cache "github.com/darkweak/souin/plugins/goyave"
	"goyave.dev/goyave/v4"
)

func main() {
	// ...
	goyave.Start(func(r *goyave.Router) {
		r.GlobalMiddleware(cache.NewHTTPCache(cache.DevDefaultConfiguration).Handle)
		// ...
	})
}
```

### Hertz middleware
To use Souin as hertz middleware, you can refer to the [Hertz middleware integration folder](https://github.com/darkweak/souin/tree/master/plugins/hertz) to discover how to configure it.  
You just have to use the `NewHTTPCache` method like below:
```go
import (
	"context"
	"net/http"

	// ...
	httpcache "github.com/darkweak/souin/plugins/hertz"
)

func main() {
	// ...
	h.Use(httpcache.NewHTTPCache(httpcache.DevDefaultConfiguration))
	// ...
}
```

### Kratos filter
To use Souin as Kratos filter, you can refer to the [Kratos plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/kratos) to discover how to configure it.  
You just have to start the Kratos HTTP server with the Souin filter like below:
```go
import (
	httpcache "github.com/darkweak/souin/plugins/kratos"
	kratos_http "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
	kratos_http.NewServer(
		kratos_http.Filter(
			httpcache.NewHTTPCacheFilter(httpcache.DevDefaultConfiguration),
		),
	)
}
```

You can also use the configuration file to configuration the HTTP cache. Refer to the code block below:
```
server: #...
data: #...
# HTTP cache part
httpcache:
  api:
    souin: {}
  default_cache:
    regex:
      exclude: /excluded
    ttl: 5s
  log_level: debug
```
After that you have to edit your server instanciation to use the HTTP cache configuration parser
```go
import (
	httpcache "github.com/darkweak/souin/plugins/kratos"
	kratos_http "github.com/go-kratos/kratos/v2/transport/http"
)

func main() {
  c := config.New(
		config.WithSource(file.NewSource("examples/configuration.yml")),
		config.WithDecoder(func(kv *config.KeyValue, v map[string]interface{}) error {
			return yaml.Unmarshal(kv.Value, v)
		}),
	)
	if err := c.Load(); err != nil {
		panic(err)
	}

	server := kratos_http.NewServer(
		kratos_http.Filter(
			httpcache.NewHTTPCacheFilter(httpcache.ParseConfiguration(c)),
		),
	)
  // ...
}
```

### Roadrunner middleware
To use Souin as Roadrunner middleware, you can refer to the [Roadrunner plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/roadrunner) to discover how to configure it.  
Ysou have to build your `rr` binary with the souin dependency.
```toml
[velox]
build_args = ['-trimpath', '-ldflags', '-s -X github.com/roadrunner-server/roadrunner/v2/internal/meta.version=${VERSION} -X github.com/roadrunner-server/roadrunner/v2/internal/meta.buildTime=${TIME}']

[roadrunner]
ref = "master"

[github]
    [github.token]
    token = "GH_TOKEN"

    [github.plugins]
    logger = { ref = "master", owner = "roadrunner-server", repository = "logger" }
    cache = { ref = "master", owner = "darkweak", repository = "souin", folder = "/plugins/roadrunner" }
	# others ...

[log]
level = "debug"
mode = "development"
```

After that, you'll be able to set each Souin configuration key under the `http.cache` key.
```yaml
# .rr.yaml
http:
  # Other http sub keys
  cache:
    api:
      basepath: /httpcache_api
      prometheus:
        basepath: /anything-for-prometheus-metrics
      souin: {}
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
      regex:
        exclude: '/excluded'
      timeout:
        backend: 5s
        cache: 1ms
      ttl: 5s
      stale: 10s
    log_level: debug
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
  middleware:
    - cache
    # Other middlewares
```


### Skipper filter
To use Souin as skipper filter, you can refer to the [Skipper plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/skipper) to discover how to configure it.  
You just have to add to your Skipper instance the Souin filter like below:
```go
package main

import (
	souin_skipper "github.com/darkweak/souin/plugins/skipper"
	"github.com/zalando/skipper"
	"github.com/zalando/skipper/filters"
)

func main() {
	skipper.Run(skipper.Options{
		Address:       ":9090",
		RoutesFile:    "example.yaml",
		CustomFilters: []filters.Spec{souin_skipper.NewSouinFilter()}},
	)
}
```

After that you will be able to declare the httpcache filter in your eskip file.
```
hello: Path("/hello") 
  -> httpcache(`{"api":{"basepath":"/souin-api","security":{"secret":"your_secret_key","enable":true,"users":[{"username":"user1","password":"test"}]},"souin":{"security":true,"enable":true}},"default_cache":{"regex":{"exclude":"ARegexHere"},"ttl":"10s","stale":"10s"},"log_level":"INFO"}`)
  -> "https://www.example.org"
```

### Træfik plugin
To use Souin as Træfik plugin, you can refer to the [pilot documentation](https://pilot.traefik.io/plugins/6123af58d00e6cd1260b290e/souin) and the [Træfik plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/traefik) to discover how to configure it.  
You have to declare the `experimental` block in your traefik static configuration file. Keep in mind Træfik run their own interpreter and they often break any dependances (such as the yaml.v3 support).
```yaml
# anywhere/traefik.yml
experimental:
  plugins:
    souin:
      moduleName: github.com/darkweak/souin
      version: v1.6.50
```
After that you can declare either the whole configuration at once in the middleware block or by service. See the examples below.
```yaml
# anywhere/dynamic-configuration
http:
  routers:
    whoami:
      middlewares:
        - http-cache
      service: whoami
      rule: Host(`domain.com`)
  middlewares:
    http-cache:
      plugin:
        souin:
          api:
            prometheus: {}
            souin: {}
          default_cache:
            regex:
              exclude: '/test_exclude.*'
            ttl: 5s
            allowed_http_verbs:
              - GET
              - HEAD
              - POST
            default_cache_control: no-store
          log_level: debug
          urls:
            'domain.com/testing':
              ttl: 5s
            'mysubdomain.domain.com':
              ttl: 50s
              default_cache_control: public, max-age=86400
          ykeys:
            The_First_Test:
              headers:
                Content-Type: '.+'
            The_Second_Test:
              url: 'the/second/.+'
            The_Third_Test:
            The_Fourth_Test:
          surrogate_keys:
            The_First_Test:
              headers:
                Content-Type: '.+'
            The_Second_Test:
              url: 'the/second/.+'
            The_Third_Test:
            The_Fourth_Test:
```

```yaml
# anywhere/docker-compose.yml
services:
#...
  whoami:
    image: traefik/whoami
    labels:
      # other labels...
      - traefik.http.routers.whoami.middlewares=http-cache
      - traefik.http.middlewares.http-cache.plugin.souin.api.souin
      - traefik.http.middlewares.http-cache.plugin.souin.default_cache.ttl=10s
      - traefik.http.middlewares.http-cache.plugin.souin.default_cache.allowed_http_verbs=GET,HEAD,POST
      - traefik.http.middlewares.http-cache.plugin.souin.log_level=debug
```

### Tyk plugin
To use Souin as a Tyk plugin, you can refer to the [Tyk plugin integration folder](https://github.com/darkweak/souin/tree/master/plugins/tyk) to discover how to configure it.  
You have to define the use of Souin as `post` and `response` custom middleware. You can compile your own Souin integration using the `Makefile` and the `docker-compose` inside the [tyk integration directory](https://github.com/darkweak/souin/tree/master/plugins/tyk) and place your generated `souin-plugin.so` file inside your `middleware` directory.
```json
{
  "name":"httpbin.org",
  "api_id":"3",
  "org_id":"3",
  "use_keyless": true,
  "version_data": {
    "not_versioned": true,
    "versions": {
      "Default": {
        "name": "Default",
        "use_extended_paths": true
      }
    }
  },
  "custom_middleware": {
    "pre": [],
    "post": [
      {
        "name": "SouinRequestHandler",
        "path": "/opt/tyk-gateway/middleware/souin-plugin.so"
      }
    ],
    "post_key_auth": [],
    "auth_check": {
      "name": "",
      "path": "",
      "require_session": false
    },
    "response": [
      {
        "name": "SouinResponseHandler",
        "path": "/opt/tyk-gateway/middleware/souin-plugin.so"
      }
    ],
    "driver": "goplugin",
    "id_extractor": {
      "extract_from": "",
      "extract_with": "",
      "extractor_config": {}
    }
  },
  "proxy":{
    "listen_path":"/httpbin/",
    "target_url":"http://httpbin.org/",
    "strip_listen_path":true
  },
  "active":true,
  "config_data": {
    "httpcache": {
      "api": {
        "souin": {
          "enable": true
        }
      },
      "cdn": {
        "api_key": "XXXX",
        "provider": "fastly",
        "strategy": "soft"
      },
      "default_cache": {
        "ttl": "5s"
      }
    }
  }
}
```

### Webgo middleware
To use Souin as webgo middleware, you can refer to the [Webgo middleware integration folder](https://github.com/darkweak/souin/tree/master/plugins/webgo) to discover how to configure it.  
You just have to define a new webgo router and tell to the instance to use the process method like below:
```go
import (
	"net/http"

	"github.com/bnkamalesh/webgo/v6"
	cache "github.com/darkweak/souin/plugins/webgo"
)

func main(){

    // ...
	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	router.Use(httpcache.Middleware)
    // ...

}
```

## Credits

Thanks to these users for contributing or helping this project in any way  
* [Oxodao](https://github.com/oxodao)
* [Deuchnord](https://github.com/deuchnord)
* [Vincent Jordan](https://github.com/vejipe)
* [Mohammed Al Sahaf](https://github.com/mohammed90)
* [Hussam Almarzooq](https://github.com/hussam-almarzoq)
* [Sata51](https://github.com/sata51)
* [Pierre Diancourt](https://github.com/pierrediancourt)
* [Burak Sezer](https://github.com/buraksezer)
* [Luc Michalski](https://github.com/lucmichalski)
* [Jenaye](https://github.com/jenaye)
* [Brennan Kinney](https://github.com/polarathene)
* [Agneev](https://github.com/agneevX)
* [Eidenschink](https://github.com/eidenschink/)
* [Massimiliano Cannarozzo](https://github.com/maxcanna/)
* [Kevin Pollet](https://github.com/kevinpollet)
* [Choelzl](https://github.com/choelzl)
* [Menci](https://github.com/menci)
* [Duy Nguyen](https://github.com/duy-nguyen-devops)
* [Kiss Karoly](https://github.com/kresike)
* [Matthias von Bargen](https://github.com/mattvb91)
* [Fred Liang](https://github.com/fredliang44)
* [Kiril Vladimirov](https://github.com/vladimiroff)
* [Viktor Szépe](https://github.com/szepeviktor)
* [Omar Ramadan](https://github.com/kkroo)
* [Jonasengelmann](https://github.com/jonasengelmann)
* [JacquesDurand](https://github.com/jacquesdurand)
* [Frederic Houle](https://github.com/frederichoule)
* [Valery Piashchynski](https://github.com/rustatian)
* [Ivan Derda](https://github.com/HobMartin)
* [p0358](https://github.com/p0358)
* [Alberto Tablado](https://github.com/aluki)
* [Yong Zhang](https://github.com/yongzhang)
