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
