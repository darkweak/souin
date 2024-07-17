+++
weight = 409
title = "Go-redis"
icon = "home_storage"
description = "Redis is an in-memory database that persists on disk"
tags = ["Beginners"]
+++

## What is Go-redis
{{% alert %}}
The go-redis client instance must connect to an external service (redis service) that you run on your own.
{{% /alert %}}

Redis is often referred to as a data structures server. What this means is that Redis provides access to mutable data structures via a set of commands, which are sent using a server-client model with TCP sockets and a simple protocol. So different processes can query and modify the same data structures in a shared way.

## Github repository
[https://github.com/redis/go-redis](https://github.com/redis/go-redis)

## Use Go-redis
### With Caddy
You have to build your caddy instance including `Souin` and `Go-redis` using `xcaddy` ([refer to the build caddy section]({{% relref "/docs/middlewares/caddy#build-your-caddy-binary" %}})).
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/go-redis/caddy
```
You will be able to use redis in your Caddyfile or JSON configuration file.
```caddyfile
{
    cache {
        ttl 1h
        redis {
            configuration {
                Addrs 127.0.0.1:6379
                DB 1
            }
        }
    }
}

route {
    cache {
        redis {
            url 192.168.1.2:6379
        }
    }
    respond "Hello HTTP cache"
}
```

## Configuration
You can find the configuration for Redis [here](https://github.com/redis/go-redis/blob/master/options.go#L31) or check the values table below.

### Values
{{< table "table-hover" >}}
| Key name              | type     | required |
|-----------------------|----------|----------|
| Addrs                 | []string | ✅       |
| DB                    | int      | ❌       |
| MaxActiveConns        | int      | ❌       |
| MaxIdleConns          | int      | ❌       |
| MaxRedirects          | int      | ❌       |
| MaxRetries            | int      | ❌       |
| MinIdleConns          | int      | ❌       |
| PoolSize              | int      | ❌       |
| Protocol              | int      | ❌       |
| ClientName            | string   | ❌       |
| IdentitySuffix        | string   | ❌       |
| MasterName            | string   | ❌       |
| Password              | string   | ❌       |
| SentinelUsername      | string   | ❌       |
| SentinelPassword      | string   | ❌       |
| Username              | string   | ❌       |
| ContextTimeoutEnabled | bool     | ❌       |
| DisableIndentity      | bool     | ❌       |
| PoolFIFO              | bool     | ❌       |
| ReadOnly              | bool     | ❌       |
| RouteByLatency        | bool     | ❌       |
| RouteRandomly         | bool     | ❌       |
| ConnMaxIdleTime       | Duration | ❌       |
| ConnMaxLifetime       | Duration | ❌       |
| DialTimeout           | Duration | ❌       |
| MaxRetryBackoff       | Duration | ❌       |
| MinRetryBackoff       | Duration | ❌       |
| PoolTimeout           | Duration | ❌       |
| ReadTimeout           | Duration | ❌       |
| WriteTimeout          | Duration | ❌       |
{{< /table >}}
