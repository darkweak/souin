+++
weight = 200
title = "Quickstart"
icon = "rocket_launch"
description = "Run and try the HTTP cache in few minutes"
tags = ["Beginners"]
+++

## Disclaimer
{{% alert icon=" " %}}
The standalone server is deprecated but still maintained for testing purpose you should not use it in production. Running Souin as [caddy module]({{% relref "/docs/middlewares/caddy" %}}) is the recommended way.
{{% /alert %}}

## Docker
The easiest way to try the HTTP cache is to run the docker container image. Simply write a `docker-compose` yaml file with a souin and a træfik services.

```yaml
# your-souin-instance/docker-compose.yml
version: '3'

x-networks: &networks
  networks:
    - your_network

services:
  souin:
    image: darkweak/souin:latest
    ports:
      - 80:80
      - 443:443
    environment:
      GOPATH: /app
    volumes:
      - /anywhere/traefik.json:/ssl/traefik.json
      - /anywhere/configuration.yml:/configuration/configuration.yml

  traefik:
    image: traefik:v3.0
    command: --providers.docker
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /anywhere/traefik.json:/acme.json

networks:
  your_network:
    external: true
```

You have write the Souin configuration YAML file.  
Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).
```yaml
default_cache:
  ttl: 10s
reverse_proxy_url: 'http://traefik' # If it's in the same network you can use http://your-service, otherwise just use https://yourdomain.com
```

## From source
If you want to run the Souin standalone server from sources, you should clone the Souin repository and run the golang standalone server.
```bash
git clone https://github.com/darkweak/souin
cd souin/plugins/souin
```

You can edit the configuration located at `plugins/souin/configuration` to configure the HTTP cache server.  
Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).  
Then you can run the server.
```bash
go run main.go
```

With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.

Simply run the following curl command to ensure everything is setup. (we ensure that `domain.com` is a local domain).
```bash
curl -i http://domain.com
```