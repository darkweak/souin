Kratos middleware: Souin
================================

This is a distributed HTTP cache module for Kratos based on [Souin](https://github.com/darkweak/souin) cache.  

## Features

 * [RFC 7234](https://httpwg.org/specs/rfc7234.html) compliant HTTP Cache.
 * Sets [the `Cache-Status` HTTP Response Header](https://httpwg.org/http-extensions/draft-ietf-httpbis-cache-header.html)
 * REST API to purge the cache and list stored resources.
 * Builtin support for distributed cache.
 * Tag-based invalidation.


## Example
There is the example about the Souin initialization.
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
With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.  
You have to pass a Kratos `Configuration` structure into the `New` method (you can use the `DefaultConfiguration` variable to have a built-in production ready configuration).  
See the full detailled configuration names [here](https://github.com/darkweak/souin#optional-configuration).

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


Other resources
---------------
You can find an example for a docker-compose stack inside the `examples` folder.  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [development kratos middleware](https://github.com/darkweak/souin/blob/master/plugins/kratos)  
