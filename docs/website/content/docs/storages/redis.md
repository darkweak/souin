+++
weight = 406
title = "Redis"
icon = "home_storage"
description = "Redis is an in-memory database that persists on disk"
tags = ["Beginners"]
+++

## What is Redis
{{% alert %}}
The redis client instance must connect to an external service (redis service) that you run on your own.
{{% /alert %}}

Redis is often referred to as a data structures server. What this means is that Redis provides access to mutable data structures via a set of commands, which are sent using a server-client model with TCP sockets and a simple protocol. So different processes can query and modify the same data structures in a shared way.

## Github repository
[https://github.com/redis/go-redis](https://github.com/redis-io/redis)

## Configuration
You can find the configuration for Redis [here](https://github.com/redis/go-redis/blob/master/options.go#L31) or check the values table below.

### Values
{{< table "table-hover" >}}
| Key name              | type          | required |
|-----------------------|---------------|----------|
| Network               | string        | ✅       |
| Addr                  | string        | ✅       |
| ClientName            | string        | ❌       |
| Protocol              | int           | ❌       |
| Username              | string        | ❌       |
| Password              | string        | ❌       |
| DB                    | int           | ❌       |
| MaxRetries            | int           | ❌       |
| MinRetryBackoff       | Duration      | ❌       |
| MaxRetryBackoff       | Duration      | ❌       |
| DialTimeout           | Duration      | ❌       |
| ReadTimeout           | Duration      | ❌       |
| WriteTimeout          | Duration      | ❌       |
| ContextTimeoutEnabled | bool          | ❌       |
| PoolFIFO              | bool          | ❌       |
| PoolSize              | int           | ❌       |
| PoolTimeout           | Duration      | ❌       |
| MinIdleConns          | int           | ❌       |
| MaxIdleConns          | int           | ❌       |
| ConnMaxIdleTime       | Duration      | ❌       |
| ConnMaxLifetime       | Duration      | ❌       |
{{< /table >}}
