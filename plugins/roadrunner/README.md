Roadrunner middleware: Souin
================================

This is a distributed HTTP cache module for Roadrunner based on [Souin](https://github.com/darkweak/souin) cache.  

## Features

 * [RFC 7234](https://httpwg.org/specs/rfc7234.html) compliant HTTP Cache.
 * Sets [the `Cache-Status` HTTP Response Header](https://httpwg.org/http-extensions/draft-ietf-httpbis-cache-header.html)
 * REST API to purge the cache and list stored resources.
 * Builtin support for distributed cache.
 * Tag-based invalidation.

## Build the roadrunner binary
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

## Example configuration
You can set each Souin configuration key under the `http.cache` key. There is a configuration example below.
```yaml
# .rr.yaml
http:
  # Other http sub keys
  cache:
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
        headers: # Add headers to the key
          - Authorization # Add the header value in the key
          - Content-Type # Add the header value in the key
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
        headers: # Add headers to the key
          - Authorization # Add the header value in the key
          - Content-Type # Add the header value in the key
        hide: true # Prevent the key from being exposed in the `Cache-Status` HTTP response header
      etcd: # If distributed is set to true, you'll have to define either the etcd or olric section
        configuration: # Configure directly the Etcd client
          endpoints: # Define multiple endpoints
            - etcd-1:2379 # First node
            - etcd-2:2379 # Second node
            - etcd-3:2379 # Third node
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
  middleware:
    - cache
    # Other middlewares
```


Other resources
---------------
You can find an example for a docker-compose stack inside the `examples` folder.  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [development roadrunner middleware](https://github.com/darkweak/souin/blob/master/plugins/roadrunner)  

