Caddy Module: http.handlers.cache
================================

This is a distributed HTTP cache module for Caddy based on [Souin](https://github.com/darkweak/souin) cache.

## Features

 * Supports most HTTP cache headers defined in [RFC 7234](https://httpwg.org/specs/rfc7234.html) (see the TODO section for limitations)
 * Sets [the `Cache-Status` HTTP Response Header](https://httpwg.org/http-extensions/draft-ietf-httpbis-cache-header.html)
 * REST API to be able to purge the cache and list stored keys.
 * Embedded (or not) distributed cache can be used and a non-distributed one is supported too.


## Example Configurations

You can find an example for the [Caddyfile](Caddyfile) or the [JSON file](configuration.json).  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and his associated [Caddyfile](https://github.com/darkweak/souin/plugins/caddy/Caddyfile)  


## TODO

* [ ] Improve the API and add relevant endpoints
