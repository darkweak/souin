+++
weight = 405
title = "Nats"
icon = "home_storage"
description = "Nats is an in-memory/filesystem storage system"
tags = ["Beginners"]
+++

## What is Nats
NATS is a simple, secure and performant communications system for digital systems, services and devices. NATS is part of the Cloud Native Computing Foundation (CNCF).

NATS has over 40 client language implementations, and its server can run on-premise, in the cloud, at the edge, and even on a Raspberry Pi. NATS can secure and simplify design and operation of modern distributed systems.

## Github repository
[https://github.com/nats-io/nats-server](https://github.com/nats-io/nats-server)

## Use Nats
### With Caddy
You have to build your caddy instance including `Souin` and `Nats` using `xcaddy` ([refer to the build caddy section]({{% relref "/docs/middlewares/caddy#build-your-caddy-binary" %}})).
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/nats/caddy
```
You will be able to use nats in your Caddyfile or JSON configuration file.
```caddyfile
{
    cache {
        ttl 1h
        nats {
            url nats://nats:4222
        }
    }
}

route {
    cache
    respond "Hello HTTP cache"
}
```

## Configuration
You can find the configuration for Nats [here](https://github.com/nats-io/nats.go/blob/main/nats.go#L267) or check the values table below.


### Values
{{< table "table-hover" >}}
| Key name                    | type     | required |
|-----------------------------|----------|----------|
| Url                         | string   | ✅       |
| Servers                     | []string | ✅       |
| MaxReconnect                | int      | ❌       |
| MaxPingsOut                 | int      | ❌       |
| ReconnectBufSize            | int      | ❌       |
| SubChanLen                  | int      | ❌       |
| Name                        | string   | ❌       |
| Nkey                        | string   | ❌       |
| User                        | string   | ❌       |
| Password                    | string   | ❌       |
| Token                       | string   | ❌       |
| ProxyPath                   | string   | ❌       |
| InboxPrefix                 | string   | ❌       |
| ReconnectWait               | Duration | ❌       |
| ReconnectJitter             | Duration | ❌       |
| ReconnectJitterTLS          | Duration | ❌       |
| Timeout                     | Duration | ❌       |
| DrainTimeout                | Duration | ❌       |
| FlusherTimeout              | Duration | ❌       |
| PingInterval                | Duration | ❌       |
| NoRandomize                 | bool     | ❌       |
| NoEcho                      | bool     | ❌       |
| Verbose                     | bool     | ❌       |
| Pedantic                    | bool     | ❌       |
| Secure                      | bool     | ❌       |
| TLSHandshakeFirst           | bool     | ❌       |
| AllowReconnect              | bool     | ❌       |
| UseOldRequestStyle          | bool     | ❌       |
| NoCallbacksAfterClientClose | bool     | ❌       |
| RetryOnFailedConnect        | bool     | ❌       |
| Compression                 | bool     | ❌       |
| IgnoreAuthErrorAbort        | bool     | ❌       |
| SkipHostLookup              | bool     | ❌       |
{{< /table >}}
