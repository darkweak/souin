<p align="center"><a href="https://github.com/darkweak/souin"><img src="docs/img/logo.svg?sanitize=true" alt="Souin logo"></a></p>

# Souin Table of Contents
1. [Souin reverse-proxy cache](#project-description)
2. [Environment variables](#environment-variables)  
  2.1. [Required variables](#required-variables)  
  2.2. [Optional variables](#optional-variables)
3. [Diagrams](#diagrams)  
  3.1. [Sequence diagram](#sequence-diagram)
4. [Cache systems](#cache-systems)
5. [Examples](#examples)  
  5.1. [Træfik container](#træfik-container)
6. [SSL](#ssl)  
  6.1. [Træfik](#træfik)

[![Travis CI](https://travis-ci.com/Darkweak/Souin.svg?branch=master)](https://travis-ci.com/Darkweak/Souin)

# <img src="docs/img/logo.svg?sanitize=true" alt="Souin logo" width="30" height="30">ouin reverse-proxy cache

## Project description
Souin is a new cache system suitable for every reverse-proxy. It will be placed on top of your current reverse-proxy whether it's Apache, Nginx or Traefik.  
As it's written in go, it can be deployed on any server and thanks docker integration, it will be easy to install it on top of a Swarm or a kubernetes instance.

## Environment variables

### Required variables
|  Variable  |  Description  |  Value example  |
|:---:|:---:|:---:|
|`CACHE_PORT`|The HTTP port Souin will be listening on |`80`|
|`CACHE_TLS_PORT`|The TLS port Souin will be listening on|`443`|
|`REDIS_URL`|The redis instance URL|- `http://redis` (Container way)<br/>`http://localhost:6379` (Local way)|
|`TTL`|Duration to cache request (in seconds)|10|
|`REVERSE_PROXY`|The reverse-proxy's instance URL (Apache, Nginx, Træfik...)|- `http://yourservice` (Container way)<br/>`http://localhost:81` (Local way)|

### Optional variables
|  Variable  |  Description  |  Value example  |
|:---:|:---:|:---:|
|`REGEX`|The regex that matches URLs not to store in cache|`http://domain.com/mypath`|

## Diagrams

### Sequence diagram
<img src="docs/plantUML/sequenceDiagram.svg?sanitize=true" alt="Sequence diagram">

## Cache systems
The cache system sits on two providers at the moment. It provides an in-memory and redis cache systems because setting, getting, updating and deleting keys in Redis is as easy as it gets.  
In order to do that, Redis could be on the same network than the Souin instance when using docker-compose or over the internet then it will use by default in-memory to avoid network latency as much as possible. 
Souin will return at first the in-memory response when it gives a non-empty response, then the redis one will be used with same condition, or fallback to the reverse proxy otherwise.

### Cache invalidation
The cache invalidation is made for CRUD requests, if you're doing a GET HTTP request, it will serve the cached response when it exists, otherwise the reverse-proxy response will be served.  
If you're doing a POST, PUT, PATCH or DELETE HTTP request, the related cache GET request will be dropped and the list endpoint will be dropped too  
It works very well with plain [API Platform](https://api-platform.com) integration (not for custom actions for now) and CRUD routes.

## Examples

### Træfik container
[Træfik](https://traefik.io) is a modern reverse-proxy and help you to manage full container architecture projects.

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
      - /anywhere/traefik.json:/acme.json
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
      - ./cmd:/app/cmd
      - /anywhere/traefik.json:/ssl/traefik.json
    <<: *networks

  redis:
    image: redis:alpine
    <<: *networks

networks:
  your_network:
    external: true
```

## SSL

### Træfik
As Souin is compatible with Træfik, it can use (and it should use) `traefik.json` provided on træfik. Souin will get new/updated certs from Træfik, then your SSL certs will be up to date as far as Træfik will be too
To provide, acme, use just have to map volume as above
```yaml
    volumes:
      - /anywhere/traefik.json:/ssl/traefik.json
```
### Apache
Souin will listen `apache.json` file. You have to setup your own way to aggregate your SSL cert files and keys. Alternatively you can use a side project called [dob](https://github.com/darkweak/dob), it's open-source and written in go too
```yaml
    volumes:
      - /anywhere/apache.json:/ssl/apache.json
```
### Nginx
Souin will listen `nginx.json` file. You have to setup your own way to aggregate your SSL cert files and keys. Alternatively you can use a side project called [dob](https://github.com/darkweak/dob), it's open-source and written in go too
```yaml
    volumes:
      - /anywhere/nginx.json:/ssl/nginx.json
```
At the moment you can't choose the path for the `*.json` in the container, they have to be placed in the `/ssl` folder. In the future you'll be able to do that just setting one env var
If none `*.json` is provided to container, a default cert will be served.


## Credits

Thanks to these users for contributing or helping this project in any way  
* [oxodao](https://github.com/oxodao)
* [Deuchnord](https://github.com/deuchnord)
* [Sata51](https://github.com/sata51)
* [Pierre Diancourt](https://github.com/pierrediancourt)
