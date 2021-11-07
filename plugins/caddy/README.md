Caddy Module: http.handlers.cache
================================

This is a distributed HTTP cache module for Caddy based on [Souin](https://github.com/darkweak/souin) cache.  

## Features

 * [RFC 7234](https://httpwg.org/specs/rfc7234.html) compliant HTTP Cache.
 * Sets [the `Cache-Status` HTTP Response Header](https://httpwg.org/http-extensions/draft-ietf-httpbis-cache-header.html)
 * REST API to purge the cache and list stored resources.
 * Builtin support for distributed cache.


## Example Configurations

You can find an example for the [Caddyfile](Caddyfile) or the [JSON file](configuration.json).  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [Caddyfile](https://github.com/darkweak/souin/blob/master/plugins/caddy/Caddyfile)  


## TODO

* [ ] Improve the API and add relevant endpoints
