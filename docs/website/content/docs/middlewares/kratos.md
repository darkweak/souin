+++
weight = 512
title = "Kratos"
icon = "extension"
description = "Use Souin directly in the Kratos web server"
tags = ["Beginners", "Advanced"]
+++

## Usage

### Directly in your code
You can enable and configure the HTTP cache directly in your golang codebase project.
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
You have to pass a Souin `BaseConfiguration` structure into the `NewHTTPCache` method (you can use the `DefaultConfiguration` variable to have a built-in production ready configuration).

### Using kratos configuration
You can configure the HTTP cache behavior through your Kratos configuration file.  
```yaml
# /somewhere/kratos-configuration.yaml
server: #...
data: #...
# HTTP cache part
httpcache:
  default_cache:
	ttl: 5s
	default_cache_control: public
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

With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.

Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).

Other resources
---------------
You can find an example for a docker-compose stack inside the [`examples` folder on the Github repository](https://github.com/darkweak/souin/tree/master/plugins/kratos/examples).
Look at the [`BaseConfiguration` structure on pkg.go.dev documentation](https://pkg.go.dev/github.com/darkweak/souin/pkg/middleware#BaseConfiguration).
