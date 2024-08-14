+++
weight = 513
title = "Roadrunner"
icon = "extension"
description = "Use Souin directly in the Roadrunner web server"
tags = ["Beginners", "Advanced"]
+++

## Build the roadrunner binary
First you need to build your roadrunner instance with the cache dependency. You should use [velox](https://github.com/roadrunner-server/velox) for that.

Define a `configuration.toml` file to tell velox what and how it must build.
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

## Configuration
You can set each Souin configuration key under the `http.cache` key. There is a configuration example below.
```yaml
# /somewhere/.rr.yaml
http:
  # Other http sub keys
  cache:
    default_cache:
      stale: 1000s
      timeout:
        backend: 10s
        cache: 20ms
      ttl: 1000s
      default_cache_control: no-store
    log_level: INFO
  middleware:
    - cache
    # Other middlewares
```

Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).

With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.

Other resources
---------------
You can find an example for a docker-compose stack inside the [`examples` folder on the Github repository](https://github.com/darkweak/souin/tree/master/plugins/roadrunner/examples).
