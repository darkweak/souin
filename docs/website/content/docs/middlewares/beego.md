+++
weight = 501
title = "Beego"
icon = "extension"
description = "Use Souin directly in the Beego web server"
tags = ["Beginners", "Advanced"]
+++

## Usage
Here is the example about the Souin initialization.
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

## Configuration

You will be able to configure the HTTP cache behavior through your Beego configuration file.  
```yaml
# /somewhere/beego-configuration.yaml
appname: beepkg
httpaddr: 127.0.0.1
httpport: 9090
runmode: dev
autorender: false
recoverpanic: false
viewspath: myview
dev:
    httpport: 8080
prod:
    httpport: 8080
test:
    httpport: 8080
httpcache:
    default_cache:
        ttl: 5s
        default_cache_control: public
    log_level: debug
```

With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.

Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).

Other resources
---------------
You can find an example for a docker-compose stack inside the [`examples` folder on the Github repository](https://github.com/darkweak/souin/tree/master/plugins/beego/examples).