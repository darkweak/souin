Goyave middleware: Souin
================================

This is a distributed HTTP cache module for Goyave based on [Souin](https://github.com/darkweak/souin) cache.  

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
With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.  
You have to pass a Goyave `Configuration` structure into the `NewHTTPCache` method (you can use the `DefaultConfiguration` variable to have a built-in production ready configuration).  
See the full detailled configuration names [here](https://github.com/darkweak/souin#optional-configuration).

Other resources
---------------
You can find an example for a docker-compose stack inside the `examples` folder.  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [development goyave middleware](https://github.com/darkweak/souin/blob/master/plugins/goyave)  
