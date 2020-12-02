<p align="center"><a href="https://github.com/darkweak/souin"><img src="docs/img/logo.svg?sanitize=true" alt="Souin logo"></a></p>

# Souin Table of Contents
1. [Souin reverse-proxy cache](#project-description)
2. [Configuration](#configuration)  
  2.1. [Required configuration](#required-configuration)  
  2.2. [Optional configuration](#optional-configuration)
3. [APIs](#apis)  
  3.1. [Souin API](#souin-api)  
  3.2. [Security API](#security-api)
4. [Diagrams](#diagrams)  
  4.1. [Sequence diagram](#sequence-diagram)
5. [Cache systems](#cache-systems)
6. [Examples](#examples)  
  6.1. [Træfik container](#træfik-container)
7. [SSL](#ssl)  
  7.1. [Træfik](#træfik)  
  7.2. [Apache](#apache)  
  7.3. [Nginx](#nginx)  
8. [Credits](#credits)

[![Travis CI](https://travis-ci.com/Darkweak/Souin.svg?branch=master)](https://travis-ci.com/Darkweak/Souin)

# <img src="docs/img/logo.svg?sanitize=true" alt="Souin logo" width="30" height="30">ouin reverse-proxy cache

## Project description
Souin is a new cache system suitable for every reverse-proxy. It will be placed on top of your current reverse-proxy whether it's Apache, Nginx or Traefik.  
As it's written in go, it can be deployed on any server and thanks to the docker integration, it will be easy to install on top of a Swarm, or a kubernetes instance.  
It's RFC compatible, supporting Vary, request coalescing and other specifications related to the [RFC-7234](https://tools.ietf.org/html/rfc7234)

## Disclaimer
If you need redis, olric or other custom cache providers, you have to use the full-featured version. You can read the documentation, on [the full-featured branch](https://github.com/Darkweak/Souin/tree/full-version) to discover the specific parts.

## Configuration
The configuration file is stored at `/anywhere/configuration.yml`. You can edit it provided you fill at least the required parameters as shown below.

### Required configuration
```yaml
default_cache: # Required part
  port: # Ports to expose Souin
    web: 80
    tls: 443
  ttl: 10 # Default TTL
reverse_proxy_url: 'http://traefik' # If it's in the same network you can use http://your-service. Then just use https://yourdomain.com
```
This is a fully working minimal configuration for a Souin instance

|  Key                           |  Description                                                       |  Value example                                                                                                            |
|:------------------------------:|:------------------------------------------------------------------:|:-------------------------------------------------------------------------------------------------------------------------:|
| `default_cache.port.{web,tls}` | The device's local HTTP/TLS port that Souin should be listening on | Respectively `80` and `443`                                                                                               |
| `default_cache.ttl`            | Duration to cache request (in seconds)                             | 10                                                                                                                        |
| `reverse_proxy_url`            | The reverse-proxy's instance URL (Apache, Nginx, Træfik...)        | - `http://yourservice` (Container way)<br/>`http://localhost:81` (Local way)<br/>`http://yourdomain.com:81` (Network way) |

### Optional configuration
```yaml
# /anywhere/configuration.yml
api:
  basepath: /souin-api # Default route basepath for every additional APIs to avoid conflicts with existing routes
  security: # Secure your APIs
    secret: your_secret_key # JWT secret key
    enable: true # Required to enable the endpoints
    users: # Users declaration
      - username: user1
        password: test
  souin: # Souin listing keys and cache management
    security: true # Enable JWT Authentication token
    enable: true # Enable the endpoints
default_cache:
  headers: # Default headers concatenated in stored keys
    - Authorization
  cache_providers:
    - all # Enable all providers by default
  redis: # Redis configuration
    url: 'redis:6379'
  regex:
    exclude: 'ARegexHere' # Regex to exclude from cache
ssl_providers: # The {providers}.json to use
  - traefik
urls:
  'https:\/\/domain.com\/first-.+': # First regex route configuration
    ttl: 1000 # Override default TTL
  'https:\/\/domain.com\/second-route': # Second regex route configuration
    ttl: 10 # Override default TTL
    headers: # Override default headers
    - Authorization
  'https?:\/\/mysubdomain\.domain\.com': # Third regex route configuration
    ttl: 50
    headers: # Override default headers
    - Authorization
    - 'Content-Type'
```

|  Key                                 |  Description                                                               |  Value example                                                               |
|:------------------------------------:|:--------------------------------------------------------------------------:|:----------------------------------------------------------------------------:|
| `api.basepath`                       | BasePath for all APIs to avoid conflicts                                   | `/your-non-conflicting-route`<br/><br/>`(default: /souin-api)`               |
| `api.{api}.enable`                   | Enable the new API with related routes                                     | `true`<br/><br/>`(default: false)`                                           |
| `api.security.secret`                | JWT secret key                                                             | `Any_charCanW0rk123`                                                         |
| `api.security.users`                 | Array of authorized users with username x password combo                   | `- username: admin`<br/><br/>`  password: admin`                             |
| `api.souin.security`                 | Enable JWT validation to access the resource                               | `true`<br/><br/>`(default: false)`                                           |
| `default_cache.headers`              | List of headers to include to the cache                                    | `- Authorization`<br/><br/>`- Content-Type`<br/><br/>`- X-Additional-Header` |
| `default_cache.cache_providers`      | Your providers list to cache your data, by default it will use all systems | `- all`<br/><br/>`- ristretto`<br/><br/>`- redis`                            |
| `default_cache.redis.url`            | The redis url, used if you enabled it in the provider section              | `redis:6379` (container way) and `http://yourdomain.com:6379` (network way)  |
| `default_cache.regex.exclude`        | The regex used to prevent paths being cached                               | `^[A-z]+.*$`                                                                 |
| `ssl_providers`                      | List of your providers handling certificates                               | `- traefik`<br/><br/>`- nginx`<br/><br/>`- apache`                           |
| `urls.{your url or regex}`           | List of your custom configuration depending each URL or regex              | 'https:\/\/yourdomain.com'                                                   |
| `urls.{your url or regex}.ttl`       | Override the default TTL if defined                                        | 99999                                                                        |
| `urls.{your url or regex}.headers`   | Override the default headers if defined                                    | `- Authorization`<br/><br/>`- 'Content-Type'`                                |
| `urls.{your url or regex}.providers` | Override the default providers if defined                                  | `- redis`<br/><br/>`- ristretto`                                             |

## APIs
All endpoints are accessible through the `api.basepath` configuration line or by default through `/souin-api` to avoid named route conflicts. Be sure to define an unused route to not break your existing application.

### Souin API
Souin API allow users to manage the cache.  
The base path for the souin API is `/souin`.

| Method  | Endpoint          | Description                                                                                             |
|:-------:|:-----------------:|:-------------------------------------------------------------------------------------------------------:|
| `GET`   | `/`               | List stored keys cache                                                                                  |
| `PURGE` | `/{id or regexp}` | Purge selected one or multiple items depending of the parameter which can be a specific key or a regexp |

### Security API
Security API allow users to protect other APIs with JWT authentication.  
The base path for the security API is `/authentication`.

| Method | Endpoint   | Body                                       | Headers                                                                         | Description                                                                                                            |
|:------:|:----------:|:------------------------------------------:|:-------------------------------------------------------------------------------:|:----------------------------------------------------------------------------------------------------------------------:|
| `POST` | `/login`   | `{"username":"admin", "password":"admin"}` | `['Content-Type' => 'json']`                                                    | Try to login, it returns a response which contains the cookie name `souin-authorization-token` with the JWT if succeed |
| `POST` | `/refresh` | `-`                                        | `['Content-Type' => 'json', 'Cookie' => 'souin-authorization-token=the-token']` | Purge selected one or multiple items depending of the parameter which can be a specific key or a regexp                |

## Diagrams

### Sequence diagram
See the sequence for the minimal version below
<img src="docs/plantUML/sequenceDiagram.svg?sanitize=true" alt="Sequence diagram">

## Cache systems
Supported providers
 - [Redis](https://github.com/go-redis/redis)
 - [Olric](https://github.com/buraksezer/olric)

 The cache system sits on top of three providers at the moment. It provides an in-memory, redis and Olric cache systems because setting, getting, updating and deleting keys in these providers is as easy as it gets.  
 In order to do that, Redis and Olric providers need to be either on the same network as the Souin instance when using docker-compose or over the internet, then it will use by default in-memory to avoid network latency as much as possible. 
 Souin will return at first the in-memory response when it gives a non-empty response, then the olric followed by the redis one with same condition, or fallback to the reverse proxy otherwise.
 Since 1.4.2, Souin supports [Olric](https://github.com/buraksezer/olric) to handle distributed cache.

### Cache invalidation
The cache invalidation is build for CRUD requests, if you're doing a GET HTTP request, it will serve the cached response when it exists, otherwise the reverse-proxy response will be served.  
If you're doing a POST, PUT, PATCH or DELETE HTTP request, the related cache GET request, and the list endpoint will be dropped.  
It works very well with plain [API Platform](https://api-platform.com) integration (not for custom actions at the moment) and CRUD routes.
Then it supports invalidation via [Souin API](#souin-api) to invalidate the cache programmatically.

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
    image: darkweak/souin:latest-full
    ports:
      - 80:80
      - 443:443
    depends_on:
      - redis
    environment:
      GOPATH: /app
    volumes:
      - /anywhere/traefik.json:/ssl/traefik.json
      - /anywhere/configuration.yml:/configuration/configuration.yml
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
To provide, acme, you just have to map volume as above
```yaml
    volumes:
      - /anywhere/traefik.json:/ssl/traefik.json
```
### Apache
Souin will listen to the `apache.json` file. You have to setup your own way to aggregate your SSL cert files and keys. Alternatively you can use a side project called [dob](https://github.com/darkweak/dob), it's open-source and written in go too
```yaml
    volumes:
      - /anywhere/apache.json:/ssl/apache.json
```
### Nginx
Souin will listen to the `nginx.json` file. You have to setup your own way to aggregate your SSL cert files and keys. Alternatively you can use a side project called [dob](https://github.com/darkweak/dob), it's open-source and written in go too
```yaml
    volumes:
      - /anywhere/nginx.json:/ssl/nginx.json
```
At the moment you can't choose the path for the `*.json` file in the container, they have to be placed in the `/ssl` folder. In the future you'll be able to do that by setting one env var
If none `*.json` file is provided to container, a default cert will be served.


## Credits

Thanks to these users for contributing or helping this project in any way  
* [oxodao](https://github.com/oxodao)
* [Deuchnord](https://github.com/deuchnord)
* [Sata51](https://github.com/sata51)
* [Pierre Diancourt](https://github.com/pierrediancourt)
* [Burak Sezer](https://github.com/buraksezer)
