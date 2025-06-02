+++
weight = 515
title = "Træfik"
icon = "extension"
description = "Use Souin directly in the Træfik web server"
tags = ["Beginners", "Advanced"]
+++

## Disclaimer
{{% alert context="warning" %}}
The Træfik team member break often the plugin loader because of the Yægi interpreter. We try to maintain this plugin compatible with the most of Træfik versions but cannot guarrantee that.
{{% /alert %}}

## Configuration
You will be able to configure the HTTP cache behavior through your Træfik configuration file.  
```yaml
# /somewhere/traefik-configuration.yaml
http:
  routers:
    whoami:
      middlewares:
        - http-cache
      service: whoami
      rule: Host(`domain.com`)
  middlewares:
    http-cache:
      plugin:
        souin:
          default_cache:
            ttl: 5s
            default_cache_control: no-store
          log_level: debug
```

You can also configure directly your HTTP cache instance directly in your docker service declaration.
```yaml
# /somewhere/docker-compose.yml
services:
#...
  whoami:
    image: traefik/whoami
    labels:
      # other labels...
      - traefik.http.routers.whoami.middlewares=http-cache
      - traefik.http.middlewares.http-cache.plugin.souin.api.souin
      - traefik.http.middlewares.http-cache.plugin.souin.default_cache.ttl=10s
      - traefik.http.middlewares.http-cache.plugin.souin.default_cache.allowed_http_verbs=GET,HEAD,POST
      - traefik.http.middlewares.http-cache.plugin.souin.log_level=debug
```

Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).

## Usage 
You have to declare the `experimental` block in your traefik static configuration file.
```yaml
# /somewhere/traefik.yml
experimental:
  plugins:
    souin:
      moduleName: github.com/darkweak/souin
      version: v1.7.7
```

With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.

Other resources
---------------
You can find an example for a docker-compose stack inside the [`traefik` folder on the Github repository](https://github.com/darkweak/souin/tree/master/plugins/traefik).
Look at the [`BaseConfiguration` structure on pkg.go.dev documentation](https://pkg.go.dev/github.com/darkweak/souin/pkg/middleware#BaseConfiguration).
