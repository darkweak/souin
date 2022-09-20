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
      headers:
        - Authorization
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


Other resources
---------------
You can find an example for a docker-compose stack inside the `examples` folder.  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [development roadrunner middleware](https://github.com/darkweak/souin/blob/master/plugins/roadrunner)  

