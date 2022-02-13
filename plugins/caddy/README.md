Caddy Module: http.handlers.cache
================================

This is a distributed HTTP cache module for Caddy based on [Souin](https://github.com/darkweak/souin) cache.  

## Features

 * [RFC 7234](https://httpwg.org/specs/rfc7234.html) compliant HTTP Cache.
 * Sets [the `Cache-Status` HTTP Response Header](https://httpwg.org/http-extensions/draft-ietf-httpbis-cache-header.html)
 * REST API to purge the cache and list stored resources.
 * Builtin support for distributed cache.


## Example Configurations
There is the fully configuration below
```caddy
{
    order cache before rewrite
    log {
        level debug
    }
    cache {
        api {
            basepath /some-basepath
            prometheus {
                basepath /prometheus-changed-endpoint-path
                security true
            }
            souin {
                basepath /souin-changed-endpoint-path
                security true
            }
            souin {
                security true
            }
        }
        badger {
            path the_path_to_a_file.json
            configuration {
                # Your badger configuration here
            }
        }
        cdn {
            api_key XXXX
            dynamic true
            email darkweak@protonmail.com
            hostname domain.com
            network your_network
            provider fastly
            strategy soft
            service_id 123456_id
            zone_id anywhere_zone
        }
        headers Content-Type Authorization
        log_level debug
        olric {
            url url_to_your_cluster:3320
            path the_path_to_a_file.yaml
            configuration {
                # Your badger configuration here
            }
        }
        regex {
            exclude /test2.*
        }
        stale 200s
        ttl 1000s
        default_cache_control no-store
    }
}

:4443
respond "Hello World!"

@match path /test1*
@match2 path /test2*
@matchdefault path /default
@souin-api path /souin-api*

cache @match {
    ttl 5s
}

cache @match2 {
    ttl 50s
    headers Authorization
    default_cache_control "public, max-age=86400"
}

cache @matchdefault {
    ttl 5s
}

cache @souin-api {}
```
What does these directives mean?  
|  Key                      |  Description                                                                                                                                 |  Value example                                                                                                          |
|:--------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| `api`                     | The cache-handler API cache management                                                                                                       |                                                                                                                         |
| `api.basepath`            | BasePath for all APIs to avoid conflicts                                                                                                     | `/your-non-conflict-route`<br/><br/>`(default: /souin-api)`                                                             |
| `api.prometheus.basepath` | Prometheus basepath endpoint                                                                                                                 | `/another-souin-api-route`<br/><br/>`(default: /metrics)`                                                               |
| `api.prometheus.security` | Enable JWT validation to access the prometheus endpoint                                                                                      | `(default: false)`                                                                                                      |
| `api.souin.basepath`      | Souin API basepath                                                                                                                           | `/another-souin-api-route`<br/><br/>`(default: /souin)`                                                                 |
| `api.souin.security`      | Enable JWT validation to access the souin endpoint                                                                                           | `(default: false)`                                                                                                      |
| `badger`                  | Configure the Badger cache storage                                                                                                           |                                                                                                                         |
| `badger.path`             | Configure Badger with a file                                                                                                                 | `/anywhere/badger_configuration.json`                                                                                   |
| `badger.configuration`    | Configure Badger directly in the Caddyfile or your JSON caddy configuration                                                                  | [See the Badger configuration for the options](https://dgraph.io/docs/badger/get-started/)                              |
| `cdn`                     | The CDN management, if you use any cdn to proxy your requests Souin will handle that                                                         |                                                                                                                         |
| `cdn.provider`            | The provider placed before Souin                                                                                                             | `akamai`<br/><br/>`fastly`<br/><br/>`souin`                                                                             |
| `cdn.api_key`             | The api key used to access to the provider                                                                                                   | `XXXX`                                                                                                                  |
| `cdn.dynamic`             | Enable the dynamic keys returned by your backend application                                                                                 | `true`<br/><br/>`(default: false)`                                                                                      |
| `cdn.email`               | The api key used to access to the provider if required, depending the provider                                                               | `XXXX`                                                                                                                  |
| `cdn.hostname`            | The hostname if required, depending the provider                                                                                             | `domain.com`                                                                                                            |
| `cdn.network`             | The network if required, depending the provider                                                                                              | `your_network`                                                                                                          |
| `cdn.strategy`            | The strategy to use to purge the cdn cache, soft will keep the content as a stale resource                                                   | `hard`<br/><br/>`(default: soft)`                                                                                       |
| `cdn.service_id`          | The service id if required, depending the provider                                                                                           | `123456_id`                                                                                                             |
| `cdn.zone_id`             | The zone id if required, depending the provider                                                                                              | `anywhere_zone`                                                                                                         |
| `default_cache_control`   | Set the default value of `Cache-Control` response header if not set by upstream (Souin treats empty `Cache-Control` as `public` if omitted)  | `no-store`                                                                                                              |
| `headers`                 | List of headers to include to the cache                                                                                                      | `Authorization Content-Type X-Additional-Header`                                                                        |
| `olric`                   | Configure the Olric cache storage                                                                                                            |                                                                                                                         |
| `olric.path`              | Configure Olric with a file                                                                                                                  | `/anywhere/badger_configuration.json`                                                                                   |
| `olric.configuration`     | Configure Olric directly in the Caddyfile or your JSON caddy configuration                                                                   | [See the Badger configuration for the options](https://github.com/buraksezer/olric/blob/master/cmd/olricd/olricd.yaml/) |
| `regex.exclude`           | The regex used to prevent paths being cached                                                                                                 | `^[A-z]+.*$`                                                                                                            |
| `stale`                   | The stale duration                                                                                                                           | `25m`                                                                                                                   |
| `ttl`                     | The TTL duration                                                                                                                             | `120s`                                                                                                                  |
| `log_level`               | The log level                                                                                                                                | `One of DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL it's case insensitive`                                           |

Other resources
---------------
You can find an example for the [Caddyfile](Caddyfile) or the [JSON file](configuration.json).  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [Caddyfile](https://github.com/darkweak/souin/blob/master/plugins/caddy/Caddyfile)  


## TODO

* [ ] Improve the API and add relevant endpoints
