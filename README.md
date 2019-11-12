<p align="center"><a href="https://github.com/darkweak/souin"><img src="docs/img/logo.svg?sanitize=true" alt="Souin logo"></a></p>

# Souin Table of Contents
1. [Souin reverse-proxy cache](#project-description)
2. [Environment variables](#environment-variables)  
  2.1. [Required variables](#required-variables)  
  2.2. [Optional variables](#optional-variables)
3. [Exemples](#exemples)  
  3.2. [Træfik container](#træfik-container)

# <img src="docs/img/logo.svg?sanitize=true" alt="Souin logo" width="30" height="30">ouin reverse-proxy cache

## Project description
Souin is a new cache system for every reverse-proxy. It will be placed on top of your reverse-proxy like Apache, NGinx or Traefik.  
As it's written in go, it can be deployed on any server and with docker integration, it will be easy to implement it on top of Swarm or kubernetes instance.

## Environment variables

### Required variables
|  Variable  |  Description  |  Value exemple  |
|:---:|:---:|:---:|
|`CACHE_PORT`|The HTTP port Souin will be running to|`80`|
|`CACHE_TLS_PORT`|The TLS port Souin will be running to|`443`|
|`REDIS_URL`|The redis instance URL|- `http://redis` (Container way)<br/>`http://localhost:6379` (Local way)|
|`TTL`|Duration to cache request (in seconds)|10|
|`REVERSE_PROXY`|The reverse-proxy instance URL like Apache, Nginx, Træfik, etc...|- `http://yourservice` (Container way)<br/>`http://localhost:81` (Local way)|

### Optional variables
|  Variable  |  Description  |  Value exemple  |
|:---:|:---:|:---:|
|`REGEX`|The regex to define URL to not store in cache|`http://domain.com/mypath`|

## Exemples

### Træfik container
[Træfik](https://traefik.io) is a modern reverse-proxy and help you to manage full container architecure projects.

```yaml
# your-traefik-instance/docker-compose.yml
version: '3.4'

x-networks: &networks
  networks:
    - your_network

services:
  traefik:
    image: traefik:v2.0
    ports:
      - "81:80" # Note the 81 to 80 port declaration
      - "444:443" # Note the 444 to 443 port declaration
    command: --providers.docker
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    <<: *networks

  # your other services here...

networks:
  your_network:
    external: true
```

```yaml
# your-souin-instance/docker-compose.yml
version: '3.4'

x-networks: &networks
  networks:
    - your_network

services:
  souin:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - ${CACHE_PORT}:80
      - ${CACHE_TLS_PORT}:443
    depends_on:
      - redis
    environment:
      REDIS_URL: ${REDIS_URL}
      TTL: ${TTL}
      CACHE_PORT: ${CACHE_PORT}
      CACHE_TLS_PORT: ${CACHE_TLS_PORT}
      REVERSE_PROXY: ${REVERSE_PROXY}
      REGEX: ${REGEX}
      GOPATH: /app
    volumes:
      - ./src:/app/src
      - ./cmd:/app/cmd
    <<: *networks

  redis:
    image: redis:alpine
    <<: *networks

networks:
  your_network:
    external: true
```
