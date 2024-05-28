Caddy Module: http.handlers.cache
================================

Development repository of the caddy cache-handler module.  
This is a distributed HTTP cache module for Caddy based on [Souin](https://github.com/darkweak/souin) cache.  

## Features

 * [RFC 7234](https://httpwg.org/specs/rfc7234.html) compliant HTTP Cache.
 * Sets [the `Cache-Status` HTTP Response Header](https://httpwg.org/http-extensions/draft-ietf-httpbis-cache-header.html)
 * REST API to purge the cache and list stored resources.
 * ESI tags processing (using the [go-esi package](https://github.com/darkweak/go-esi)).
 * Builtin support for distributed cache.

## Minimal Configuration
Using the minimal configuration the responses will be cached for `120s`
```caddy
{
    order cache before rewrite
    cache
}

example.com {
    cache
    reverse_proxy your-app:8080
}
```

## Global Option Syntax
Here are all the available options for the global options
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
                disable_query
                headers X-Token Authorization
                hide
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
            headers Content-Type Authorization
        }
        log_level debug
        mode bypass
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
```

## Cache directive Syntax
Here are all the available options for the directive options

```
@match path /path

handle @match {
    cache {
        cache_name ChangeName
        cache_keys {
            (host1|host2).*\.css {
                disable_body
                disable_host
                disable_method
                disable_query
                headers X-Token Authorization
            }
        }
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
        key {
            disable_body
            disable_host
            disable_method
            disable_query
            headers Content-Type Authorization
        }
        log_level debug
        regex {
            exclude /test2.*
        }
        stale 200s
        ttl 1000s
        default_cache_control no-store
    }
}
```

## Provider Syntax

### Badger
The badger provider must have either the path or the configuration directive.
```
badger-path.com {
    cache {
        badger {
            path /tmp/badger/first-match
        }
    }
}
```
```
badger-configuration.com {
    cache {
        badger {
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
                EncryptionKeyRotationDuration <Duration>
                BypassLockGuard <bool>
                ChecksumVerificationMode <int>
                DetectConflicts <bool>
                NamespaceOffset <int>
            }
        }
    }
}
```

### Etcd
The etcd provider must have the configuration directive.
```
etcd-configuration.com {
    cache {
        etcd {
            configuration {
                Endpoints etcd1:2379 etcd2:2379 etcd3:2379
                AutoSyncInterval 1s
                DialTimeout 1s
                DialKeepAliveTime 1s
                DialKeepAliveTimeout 1s
                MaxCallSendMsgSize 10000000
                MaxCallRecvMsgSize 10000000
                Username john
                Password doe
                RejectOldCluster false
                PermitWithoutStream false
            }
        }
    }
}
```

### NutsDB
The nutsdb provider must have either the path or the configuration directive.
```
nuts-path.com {
    cache {
        nuts {
            path /tmp/nuts-path
        }
    }
}
```
```
nuts-configuration.com {
    cache {
        nuts {
            configuration {
                Dir /tmp/nuts-configuration
                EntryIdxMode 1
                RWMode 0
                SegmentSize 1024
                NodeNum 42
                SyncEnable true
                StartFileLoadingMode 1
            }
        }
    }
}
```

### Olric
The olric provider must have either the url directive to work as client mode.
```
olric-url.com {
    cache {
        olric {
            url olric:3320
        }
    }
}
```

The olric provider must have either the path or the configuration directive to work as embedded mode.
```
olric-path.com {
    cache {
        olric {
            path /path/to/olricd.yml
        }
    }
}
```
```
olric-configuration.com {
    cache {
        nuts {
            configuration {
                Dir /tmp/nuts-configuration
                EntryIdxMode 1
                RWMode 0
                SegmentSize 1024
                NodeNum 42
                SyncEnable true
                StartFileLoadingMode 1
            }
        }
    }
}
```

### Redis
The redis provider must have either the URL or the configuration directive.

```
redis-url.com {
    cache {
        redis {
            url 127.0.0.1:6379
        }
    }
}
```
```
redis-configuration.com {
    cache {
        redis {
            configuration {
                Network my-network
                Addr 127.0.0.1:6379
                Username user
                Password password
                DB 1
                MaxRetries 1
                MinRetryBackoff 5s
                MaxRetryBackoff 5s
                DialTimeout 5s
                ReadTimeout 5s
                WriteTimeout 5s
                PoolFIFO true
                PoolSize 99999
                PoolTimeout 10s
                MinIdleConns 100
                MaxIdleConns 100
                ConnMaxIdleTime 5s
                ConnMaxLifetime 5s
            }
        }
    }
}
```

What does these directives mean?  
|  Key                                      |  Description                                                                                                                                 |  Value example                                                                                                          |
|:------------------------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------|
| `allowed_http_verbs`                      | The HTTP verbs allowed to be cached                                                                                                          | `GET POST PATCH`<br/><br/>`(default: GET HEAD)`                                                                         |
| `api`                                     | The cache-handler API cache management                                                                                                       |                                                                                                                         |
| `api.basepath`                            | BasePath for all APIs to avoid conflicts                                                                                                     | `/your-non-conflict-route`<br/><br/>`(default: /souin-api)`                                                             |
| `api.prometheus`                          | Enable the Prometheus metrics                                                                                                                |                                                                                                                         |
| `api.souin.basepath`                      | Souin API basepath                                                                                                                           | `/another-souin-api-route`<br/><br/>`(default: /souin)`                                                                 |
| `badger`                                  | Configure the Badger cache storage                                                                                                           |                                                                                                                         |
| `badger.path`                             | Configure Badger with a file                                                                                                                 | `/anywhere/badger_configuration.json`                                                                                   |
| `badger.configuration`                    | Configure Badger directly in the Caddyfile or your JSON caddy configuration                                                                  | [See the Badger configuration for the options](https://dgraph.io/docs/badger/get-started/)                              |
| `cache_name`                              | Override the cache name to use in the Cache-Status response header                                                                           | `Another` `Caddy` `Cache-Handler` `Souin`                                                                               |
| `cache_keys`                              | Define the key generation rules for each URI matching the key regexp                                                                         |                                                                                                                         |
| `cache_keys.{your regexp}`                | Regexp that the URI should match to override the key generation                                                                              | `.+\.css`                                                                                                               |
| `cache_keys.{your regexp}`                | Regexp that the URI should match to override the key generation                                                                              | `.+\.css`                                                                                                               |
| `cache_keys.{your regexp}.disable_body`   | Disable the body part in the key matching the regexp (GraphQL context)                                                                       | `true`<br/><br/>`(default: false)`                                                                                      |
| `cache_keys.{your regexp}.disable_host`   | Disable the host part in the key matching the regexp                                                                                         | `true`<br/><br/>`(default: false)`                                                                                      |
| `cache_keys.{your regexp}.disable_method` | Disable the method part in the key matching the regexp                                                                                       | `true`<br/><br/>`(default: false)`                                                                                      |
| `cache_keys.{your regexp}.disable_query`  | Disable the query string part in the key matching the regexp                                                                                 | `true`<br/><br/>`(default: false)`                                                                                      |
| `cache_keys.{your regexp}.headers`        | Add headers to the key matching the regexp                                                                                                   | `Authorization Content-Type X-Additional-Header`                                                                        |
| `cache_keys.{your regexp}.hide`           | Prevent the key from being exposed in the `Cache-Status` HTTP response header                                                                | `true`<br/><br/>`(default: false)`                                                                                      |
| `cdn`                                     | The CDN management, if you use any cdn to proxy your requests Souin will handle that                                                         |                                                                                                                         |
| `cdn.provider`                            | The provider placed before Souin                                                                                                             | `akamai`<br/><br/>`fastly`<br/><br/>`souin`                                                                             |
| `cdn.api_key`                             | The api key used to access to the provider                                                                                                   | `XXXX`                                                                                                                  |
| `cdn.dynamic`                             | Enable the dynamic keys returned by your backend application                                                                                 | `(default: true)`                                                                                                       |
| `cdn.email`                               | The api key used to access to the provider if required, depending the provider                                                               | `XXXX`                                                                                                                  |
| `cdn.hostname`                            | The hostname if required, depending the provider                                                                                             | `domain.com`                                                                                                            |
| `cdn.network`                             | The network if required, depending the provider                                                                                              | `your_network`                                                                                                          |
| `cdn.strategy`                            | The strategy to use to purge the cdn cache, soft will keep the content as a stale resource                                                   | `hard`<br/><br/>`(default: soft)`                                                                                       |
| `cdn.service_id`                          | The service id if required, depending the provider                                                                                           | `123456_id`                                                                                                             |
| `cdn.zone_id`                             | The zone id if required, depending the provider                                                                                              | `anywhere_zone`                                                                                                         |
| `default_cache_control`                   | Set the default value of `Cache-Control` response header if not set by upstream (Souin treats empty `Cache-Control` as `public` if omitted)  | `no-store`                                                                                                              |
| `max_cachable_body_bytes`                 | Set the maximum size (in bytes) for a response body to be cached (unlimited if omited)                                                       | `1048576` (1MB)                                                                                                         |
| `key`                                     | Override the key generation with the ability to disable unecessary parts                                                                     |                                                                                                                         |
| `key.disable_body`                        | Disable the body part in the key (GraphQL context)                                                                                           | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.disable_host`                        | Disable the host part in the key                                                                                                             | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.disable_method`                      | Disable the method part in the key                                                                                                           | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.disable_query`                       | Disable the query string part in the key                                                                                                     | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.disable_scheme`                      | Disable the scheme string part in the key                                                                                                    | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.hash`                                | Hash the key before store it in the storage to get smaller keys                                                                              | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.headers`                             | Add headers to the key matching the regexp                                                                                                   | `Authorization Content-Type X-Additional-Header`                                                                        |
| `key.hide`                                | Prevent the key from being exposed in the `Cache-Status` HTTP response header                                                                | `true`<br/><br/>`(default: false)`                                                                                      |
| `key.template`                            | Use caddy templates to create the key (when this option is enabled, disable_* directives are skipped)                                        | `KEY-{http.request.uri.path}-{http.request.uri.query}`                                                                  |
| `mode`                                    | Bypass the RFC respect                                                                                                                       | One of `bypass` `bypass_request` `bypass_response` `strict` (default `strict`)                                          |
| `nuts`                                    | Configure the Nuts cache storage                                                                                                             |                                                                                                                         |
| `nuts.path`                               | Set the Nuts file path storage                                                                                                               | `/anywhere/nuts/storage`                                                                                                |
| `nuts.configuration`                      | Configure Nuts directly in the Caddyfile or your JSON caddy configuration                                                                    | [See the Nuts configuration for the options](https://github.com/nutsdb/nutsdb#default-options)                          |
| `etcd`                                    | Configure the Etcd cache storage                                                                                                             |                                                                                                                         |
| `etcd.configuration`                      | Configure Etcd directly in the Caddyfile or your JSON caddy configuration                                                                    | [See the Etcd configuration for the options](https://pkg.go.dev/go.etcd.io/etcd/clientv3#Config)                        |
| `olric`                                   | Configure the Olric cache storage                                                                                                            |                                                                                                                         |
| `olric.path`                              | Configure Olric with a file                                                                                                                  | `/anywhere/olric_configuration.json`                                                                                    |
| `olric.configuration`                     | Configure Olric directly in the Caddyfile or your JSON caddy configuration                                                                   | [See the Olric configuration for the options](https://github.com/buraksezer/olric/blob/master/cmd/olricd/olricd.yaml/)  |
| `redis`                                   | Configure the Redis cache storage                                                                                                            |                                                                                                                         |
| `redis.url`                               | Set the Redis url storage                                                                                                                    | `localhost:6379`                                                                                                        |
| `redis.configuration`                     | Configure Redis directly in the Caddyfile or your JSON caddy configuration                                                                   | [See the Nuts configuration for the options](https://github.com/nutsdb/nutsdb#default-options)                          |
| `regex.exclude`                           | The regex used to prevent paths being cached                                                                                                 | `^[A-z]+.*$`                                                                                                            |
| `stale`                                   | The stale duration                                                                                                                           | `25m`                                                                                                                   |
| `timeout`                                 | The timeout configuration                                                                                                                    |                                                                                                                         |
| `timeout.backend`                         | The timeout duration to consider the backend as unreachable                                                                                  | `10s`                                                                                                                   |
| `timeout.cache`                           | The timeout duration to consider the cache provider as unreachable                                                                           | `10ms`                                                                                                                  |
| `ttl`                                     | The TTL duration                                                                                                                             | `120s`                                                                                                                  |
| `log_level`                               | The log level                                                                                                                                | `One of DEBUG, INFO, WARN, ERROR, DPANIC, PANIC, FATAL it's case insensitive`                                           |

Other resources
---------------
You can find an example for the [Caddyfile](Caddyfile) or the [JSON file](configuration.json).  
See the [Souin](https://github.com/darkweak/souin) configuration for the full configuration, and its associated [Caddyfile](https://github.com/darkweak/souin/blob/master/plugins/caddy/Caddyfile)  
