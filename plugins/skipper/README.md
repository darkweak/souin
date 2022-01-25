Skipper middleware: Souin
================================

This is a distributed HTTP cache module for Skipper based on [Souin](https://github.com/darkweak/souin) cache.  

## Features

 * [RFC 7234](https://httpwg.org/specs/rfc7234.html) compliant HTTP Cache.
 * Sets [the `Cache-Status` HTTP Response Header](https://httpwg.org/http-extensions/draft-ietf-httpbis-cache-header.html)
 * REST API to purge the cache and list stored resources.
 * Builtin support for distributed cache.
 * Tag-based invalidation.


## Example
There is the example about the Souin initialization in the Skipper eskip file.
```
hello: Path("/hello") 
  -> httpcache(`{"api":{"basepath":"/souin-api","security":{"secret":"your_secret_key","enable":true,"users":[{"username":"user1","password":"test"}]},"souin":{"security":true,"enable":true}},"default_cache":{"headers":["Authorization"],"regex":{"exclude":"ARegexHere"},"ttl":"10s","stale":"10s"},"log_level":"INFO"}`)
  -> "https://www.example.org"
```

And now the usage inside a Skipper instance.
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
With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.  
You have to pass a Souin stringified JSON `Configuration` structure into the `httpcache` filter declaration.  
See the full detailled configuration [here](https://github.com/darkweak/souin#optional-configuration).

Other resources
---------------
You can find an example for a docker-compose stack inside the `examples` folder.  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [development skipper middleware](https://github.com/darkweak/souin/blob/master/plugins/skipper)  
