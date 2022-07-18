Caddy Module: http.handlers.cache
================================

Development repository of the caddy cache-handler module.  
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
        allowed_http_verbs GET POST PATCH
        api {
            basepath /some-basepath
            prometheus
            souin {
                basepath /souin-changed-endpoint-path
            }
        }
        badger {
            path the_path_to_a_file.json
        }
        cache_keys {
            .*\.css {
                disable_body
                disable_host
                disable_method
            }
        }
        cache_name Another
        cdn {
            api_key XXXX
            dynamic
            email darkweak@protonmail.com
            hostname domain.com
            network your_network
            provider fastly
            strategy soft
            service_id 123456_id
            zone_id anywhere_zone
        }
        etcd {
            configuration {
                # Your etcd configuration here
            }
        }
        key {
            disable_body
            disable_host
            disable_method
        }
        headers Content-Type Authorization
        log_level debug
        nuts {
            path /path/to/the/storage
        }
        olric {
            url url_to_your_cluster:3320
            path the_path_to_a_file.yaml
            configuration {
                # Your olric configuration here
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
    badger {
        path /tmp/badger/first-match
        configuration {
            # Required value
            ValueDir <string>

            # Optional
            SyncWrites <bool>
            NumVersionsToKeep <int>
            ReadOnly <bool>
            Compression <int>
            InMemory <bool>
            MetricsEnabled <bool>
            MemTableSize <int>
            BaseTableSize <int>
            BaseLevelSize <int>
            LevelSizeMultiplier <int>
            TableSizeMultiplier <int>
            MaxLevels <int>
            VLogPercentile <float>
            ValueThreshold <int>
            NumMemtables <int>
            BlockSize <int>
            BloomFalsePositive <float>
            BlockCacheSize <int>
            IndexCacheSize <int>
            NumLevelZeroTables <int>
            NumLevelZeroTablesStall <int>
            ValueLogFileSize <int>
            ValueLogMaxEntries <int>
            NumCompactors <int>
            CompactL0OnClose <bool>
            LmaxCompaction <bool>
            ZSTDCompressionLevel <int>
            VerifyValueChecksum <bool>
            EncryptionKey <string>
            EncryptionKey <Duration>
            BypassLockGuard <bool>
            ChecksumVerificationMode <int>
            DetectConflicts <bool>
            NamespaceOffset <int>
        }
    }
}

cache @match2 {
    ttl 50s
    badger {
        path /tmp/badger/second-match
        configuration {
            ValueDir match2
            ValueLogFileSize 16777216
            MemTableSize 4194304
            ValueThreshold 524288
            BypassLockGuard true
        }
    }
    headers Authorization
    default_cache_control "public, max-age=86400"
}

cache @matchdefault {
    ttl 5s
    badger {
        path /tmp/badger/default-match
        configuration {
            ValueDir default
            ValueLogFileSize 16777216
            MemTableSize 4194304
            ValueThreshold 524288
            BypassLockGuard true
        }
    }
    cache_name ChangeName
    cache_keys {
        (host1|host2).*\.css {
            disable_body
            disable_host
            disable_method
        }
    }
}

cache @souin-api {}
```
What does these directives mean?  
|  Key                               |  Description                                                                                                                                 |  Value example                                                                                                          |
|:-----------------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| `allowed_http_verbs`               | The HTTP verbs allowed to be cached                                                                                                          | `GET POST PATCH`<br/><br/>`(default: GET HEAD)`                                                                         |
| `api`                              | The cache-handler API cache management                                                                                                       |                                                                                                                         |
| `api.basepath`                     | BasePath for all APIs to avoid conflicts                                                                                                     | `/your-non-conflict-route`<br/><br/>`(default: /souin-api)`                                                             |
| `api.prometheus`                   | Enable the Prometheus metrics                                                                                                                |                                                                                                                         |
| `api.souin.basepath`               | Souin API basepath                                                                                                                           | `/another-souin-api-route`<br/><br/>`(default: /souin)`                                                                 |
| `badger`                           | Configure the Badger cache storage                                                                                                           |                                                                                                                         |
| `badger.path`                      | Configure Badger with a file                                                                                                                 | `/anywhere/badger_configuration.json`                                                                                   |
| `badger.configuration`             | Configure Badger directly in the Caddyfile or your JSON caddy configuration                                                                  | [See the Badger configuration for the options](https://dgraph.io/docs/badger/get-started/)                              |
| `cache_name`                       | Override the cache name to use in the Cache-Status response header                                                                           | `Another` `Caddy` `Cache-Handler` `Souin`                                                                               |
| `cache_keys`                       | Define the key generation rules for each URI matching the key regexp                                                                         |                                                                                                                         |
| `cache_keys.{your regexp}`         | Regexp that the URI should match to override the key generation                                                                              | `.+\.css`                                                                                                               |
| `default_cache.key.disable_body`   | Disable the body part in the key matching the regexp (GraphQL context)                                                                       | `true`<br/><br/>`(default: false)`                                                                                      |
| `default_cache.key.disable_host`   | Disable the host part in the key matching the regexp                                                                                         | `true`<br/><br/>`(default: false)`                                                                                      |
| `default_cache.key.disable_method` | Disable the method part in the key matching the regexp                                                                                       | `true`<br/><br/>`(default: false)`                                                                                      |
| `cdn`                              | The CDN management, if you use any cdn to proxy your requests Souin will handle that                                                         |                                                                                                                         |
| `cdn.provider`                     | The provider placed before Souin                                                                                                             | `akamai`<br/><br/>`fastly`<br/><br/>`souin`                                                                             |
| `cdn.api_key`                      | The api key used to access to the provider                                                                                                   | `XXXX`                                                                                                                  |
| `cdn.dynamic`                      | Enable the dynamic keys returned by your backend application                                                                                 | `(default: false)`                                                                                                      |
| `cdn.email`                        | The api key used to access to the provider if required, depending the provider                                                               | `XXXX`                                                                                                                  |
| `cdn.hostname`                     | The hostname if required, depending the provider                                                                                             | `domain.com`                                                                                                            |
| `cdn.network`                      | The network if required, depending the provider                                                                                              | `your_network`                                                                                                          |
| `cdn.strategy`                     | The strategy to use to purge the cdn cache, soft will keep the content as a stale resource                                                   | `hard`<br/><br/>`(default: soft)`                                                                                       |
| `cdn.service_id`                   | The service id if required, depending the provider                                                                                           | `123456_id`                                                                                                             |
| `cdn.zone_id`                      | The zone id if required, depending the provider                                                                                              | `anywhere_zone`                                                                                                         |
| `default_cache_control`            | Set the default value of `Cache-Control` response header if not set by upstream (Souin treats empty `Cache-Control` as `public` if omitted)  | `no-store`                                                                                                              |
| `headers`                          | List of headers to include to the cache                                                                                                      | `Authorization Content-Type X-Additional-Header`                                                                        |
| `key`                              | Override the key generation with the ability to disable unecessary parts                                                                     |                                                                                                                         |
| `key.disable_body`                 | Disable the body part in the key (GraphQL context)                                                                                           | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.disable_host`                 | Disable the host part in the key                                                                                                             | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.disable_method`               | Disable the method part in the key                                                                                                           | `true`<br/><br/>`(default: false)`                                                                                      |
| `nuts`                             | Configure the Nuts cache storage                                                                                                             |                                                                                                                         |
| `nuts.path`                        | Set the Nuts file path storage                                                                                                               | `/anywhere/nuts/storage`                                                                                                |
| `nuts.configuration`               | Configure Nuts directly in the Caddyfile or your JSON caddy configuration                                                                    | [See the Nuts configuration for the options](https://github.com/nutsdb/nutsdb#default-options)                          |
| `etcd`                             | Configure the Etcd cache storage                                                                                                             |                                                                                                                         |
| `etcd.configuration`               | Configure Etcd directly in the Caddyfile or your JSON caddy configuration                                                                    | [See the Etcd configuration for the options](https://pkg.go.dev/go.etcd.io/etcd/clientv3#Config)                        |
| `olric`                            | Configure the Olric cache storage                                                                                                            |                                                                                                                         |
| `olric.path`                       | Configure Olric with a file                                                                                                                  | `/anywhere/olric_configuration.json`                                                                                    |
| `olric.configuration`              | Configure Olric directly in the Caddyfile or your JSON caddy configuration                                                                   | [See the Olric configuration for the options](https://github.com/buraksezer/olric/blob/master/cmd/olricd/olricd.yaml/)  |
| `regex.exclude`                    | The regex used to prevent paths being cached                                                                                                 | `^[A-z]+.*$`                                                                                                            |
| `stale`                            | The stale duration                                                                                                                           | `25m`                                                                                                                   |
| `ttl`                              | The TTL duration                                                                                                                             | `120s`                                                                                                                  |
| `log_level`                        | The log level                                                                                                                                | `One of DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL it's case insensitive`                                           |

Other resources
---------------
You can find an example for the [Caddyfile](Caddyfile) or the [JSON file](configuration.json).  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [Caddyfile](https://github.com/darkweak/souin/blob/master/plugins/caddy/Caddyfile)  

## TODO

* [ ] Improve the API and add relevant endpoints
