+++
weight = 509
title = "Goa"
icon = "extension"
description = "Use Souin directly in the Goa web server"
tags = ["Beginners", "Advanced"]
+++

## Usage
Here is the example about the Souin initialization.
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
With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.  
You have to pass a Souin `BaseConfiguration` structure into the `NewHTTPCache` method (you can use the `DefaultConfiguration` variable to have a built-in production ready configuration).  

Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).

Other resources
---------------
You can find an example for a docker-compose stack inside the [`examples` folder on the Github repository](https://github.com/darkweak/souin/tree/master/plugins/goa/examples).
Look at the [`BaseConfiguration` structure on pkg.go.dev documentation](https://pkg.go.dev/github.com/darkweak/souin/pkg/middleware#BaseConfiguration).
