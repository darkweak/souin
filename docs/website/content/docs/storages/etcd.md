+++
weight = 403
title = "Etcd"
icon = "home_storage"
description = "Etcd is a distributed reliable key-value store for the most critical data of a distributed system"
tags = ["Beginners"]
+++

## What is Etcd
etcd is a distributed reliable key-value store for the most critical data of a distributed system, with a focus on being:
* Simple: well-defined, user-facing API (gRPC)
* Secure: automatic TLS with optional client cert authentication
* Fast: benchmarked 10,000 writes/sec
* Reliable: properly distributed using Raft

etcd is written in Go and uses the Raft consensus algorithm to manage a highly-available replicated log.

## Github repository
[https://github.com/etcd-io/etcd](https://github.com/etcd-io/etcd)

## Use Etcd
### With Caddy
You have to build your caddy instance including `Souin` and `Etcd` using `xcaddy` ([refer to the build caddy section]({{% relref "/docs/middlewares/caddy#build-your-caddy-binary" %}})).
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/etcd/caddy
```
You will be able to use etcd in your Caddyfile or JSON configuration file.
```caddyfile
{
    cache {
        ttl 1h
        etcd
    }
}

route {
    cache
    respond "Hello HTTP cache"
}
```

## Configuration
You can find the configuration for Etcd [here](https://github.com/etcd-io/etcd/blob/main/client/v3/config.go#L28) or check the values table below.

### Values
{{< table "table-hover" >}}
| Key name           | type          | required |
|--------------------|---------------|----------|
Endpoints            | []string      | ✅       |
AutoSyncInterval     | time.Duration | ❌       |
DialTimeout          | time.Duration | ❌       |
DialKeepAliveTime    | time.Duration | ❌       |
DialKeepAliveTimeout | time.Duration | ❌       |
MaxCallSendMsgSize   | int           | ❌       |
MaxCallRecvMsgSize   | int           | ❌       |
Username             | string        | ❌       |
Password             | string        | ❌       |
RejectOldCluster     | bool          | ❌       |
PermitWithoutStream  | bool          | ❌       |         
{{< /table >}}
