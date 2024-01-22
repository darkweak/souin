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
[https://github.com/redis/rueidis](https://github.com/redis/rueidis)

## Configuration
You can find the configuration for Redis [here](https://github.com/redis/rueidis/blob/master/options.go#L31) or check the values table below.

### Values
{{< table "table-hover" >}}
| Key name              | type          | required |
|-----------------------|---------------|----------|
| InitAddress           | []string      | ✅       |
| Username              | string        | ❌       |
| Password              | string        | ❌       |
| ClientName            | string        | ❌       |
| ClientSetInfo         | []string      | ❌       |
| ClientTrackingOptions | []string      | ❌       |
| SelectDB              | int           | ❌       |
| CacheSizeEachConn     | int           | ❌       |
| RingScaleEachConn     | int           | ❌       |
| ReadBufferEachConn    | int           | ❌       |
| WriteBufferEachConn   | int           | ❌       |
| BlockingPoolSize      | int           | ❌       |
| PipelineMultiplex     | int           | ❌       |
| ConnWriteTimeout      | time.Duration | ❌       |
| MaxFlushDelay         | time.Duration | ❌       |
| ShuffleInit           | bool          | ❌       |
| ClientNoTouch         | bool          | ❌       |
| DisableRetry          | bool          | ❌       |
| DisableCache          | bool          | ❌       |
| AlwaysPipelining      | bool          | ❌       |
| AlwaysRESP2           | bool          | ❌       |
| ForceSingleClient     | bool          | ❌       |
| ReplicaOnly           | bool          | ❌       |
| ClientNoEvict         | bool          | ❌       |
{{< /table >}}
