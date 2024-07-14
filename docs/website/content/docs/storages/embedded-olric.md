+++
weight = 402
title = "Embedded Olric"
icon = "home_storage"
description = "Olric is a distributed in-memory storage system"
tags = ["Beginners"]
+++

## What is Embedded Olric
Olric is a distributed, in-memory object store. It's designed from the ground up to be distributed, and it can be used both as an embedded Go library and as a language-independent service.

With Olric, you can instantly create a fast, scalable, shared pool of RAM across a cluster of computers.

Olric is implemented in Go and uses the Redis serialization protocol. So Olric has client implementations in all major programming languages.

Olric is highly scalable and available. Distributed applications can use it for distributed caching, clustering and publish-subscribe messaging.

## Github repository
[https://github.com/buraksezer/olric](https://github.com/buraksezer/olric)

## Use Embedded Olric
### With Caddy
You have to build your caddy instance including `Souin` and `Embedded Olric` using `xcaddy` ([refer to the build caddy section]({{% relref "/docs/middlewares/caddy#build-your-caddy-binary" %}})).
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/olric/caddy
```
You will be able to use olric in your Caddyfile or JSON configuration file.
```caddyfile
{
    cache {
        ttl 1h
        olric
    }
}

route {
    cache
    respond "Hello HTTP cache"
}
```

## Configuration
You can find the configuration for the Embedded Olric [here](https://github.com/buraksezer/olric/blob/master/config/config.go#L167) or check the values table below.

### Values
{{< table "table-hover" >}}
| Key name                   | type          | required |
|----------------------------|---------------|----------|
| Interface                  | string        | ✅       |      
| LogVerbosity               | int32         | ❌       |     
| LogLevel                   | string        | ❌       |      
| BindAddr                   | string        | ❌       |      
| BindPort                   | int           | ❌       |   
| KeepAlivePeriod            | time.Duration | ❌       |             
| IdleClose                  | time.Duration | ❌       |             
| BootstrapTimeout           | time.Duration | ❌       |             
| RoutingTablePushInterval   | time.Duration | ❌       |             
| TriggerBalancerInterval    | time.Duration | ❌       |             
| Peers                      | []string      | ❌       |        
| PartitionCount             | uint64        | ❌       |      
| ReplicaCount               | int           | ❌       |   
| ReadQuorum                 | int           | ❌       |   
| WriteQuorum                | int           | ❌       |   
| MemberCountQuorum          | int32         | ❌       |     
| ReadRepair                 | bool          | ❌       |    
| ReplicationMode            | int           | ❌       |   
| LoadFactor                 | float64       | ❌       |       
| EnableClusterEventsChannel | bool          | ❌       |    
| JoinRetryInterval          | time.Duration | ❌       |             
| MaxJoinAttempts            | int           | ❌       |   
| MemberlistInterface        | string        | ❌       |      
| LeaveTimeout               | time.Duration | ❌       |             
{{< /table >}}
