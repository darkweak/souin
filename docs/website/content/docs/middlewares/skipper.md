+++
weight = 514
title = "Skipper"
icon = "extension"
description = "Use Souin directly in the Skipper web server"
tags = ["Beginners", "Advanced"]
+++

## Configuration
First you need to configure your skipper instance with the cache dependency in the eskip configuration file.  

{{% alert icon=" " %}}
The configuration is a stringified JSON, it's quite painful to write it but that's the Skipper configuration format 🤷‍♂️.
{{% /alert %}}

```yaml
# /somewhere/example.yaml
default: Path("/*") -> httpcache(`{"api":{"basepath":"/souin-api","prometheus":{"enable":true},"souin":{"security":true,"enable":true}},"default_cache":{"headers":["Authorization"],"regex":{"exclude":"/excluded"},"ttl":"5s"},"log_level":"INFO"}`) -> inlineContent("[1,2,3]", "application/json") -> <shunt>
```

Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).

## Usage
You can now use the `NewSouinFilter` in your Skipper codebase project to enable the HTTP cache. Using `RoutesFile` property will parse and use the Souin configuration defined in it to configure the HTTP cache behavior.
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


Other resources
---------------
You can find an example for a docker-compose stack inside the [`examples` folder on the Github repository](https://github.com/darkweak/souin/tree/master/plugins/skipper/examples).
