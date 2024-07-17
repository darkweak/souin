+++
weight = 406
title = "Nuts"
icon = "home_storage"
description = "Nuts is an in-memory/filesystem storage system"
tags = ["Beginners"]
+++

## What is Nuts
NutsDB is a simple, fast, embeddable and persistent key/value store written in pure Go.

It supports fully serializable transactions and many data structures such as lis, set, sorted set. All operations happen inside a Tx. Tx represents a transaction, which can be read-only or read-write. Read-only transactions can read values for a given bucket and a given key or iterate over a set of key-value pairs. Read-write transactions can read, update and delete keys from the DB.

## Github repository
[https://github.com/nutsdb/nutsdb](https://github.com/nutsdb/nutsdb)

## Use Nuts
### With Caddy
You have to build your caddy instance including `Souin` and `Nuts` using `xcaddy` ([refer to the build caddy section]({{% relref "/docs/middlewares/caddy#build-your-caddy-binary" %}})).
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/nuts/caddy
```
You will be able to use nuts in your Caddyfile or JSON configuration file.
```caddyfile
{
    cache {
        ttl 1h
        nuts
    }
}

route {
    cache
    respond "Hello HTTP cache"
}
```

## Configuration
You can find the configuration for Nuts [here](https://github.com/nutsdb/nutsdb/blob/master/options.go#L55) or check the values table below.

### Values
{{< table "table-hover" >}}
| Key name               | type              | required |
|------------------------|-------------------|----------|
| Dir                    | string            | ✅       |
| EntryIdxMode           | EntryIdxMode      | ❌       |
| RWMode                 | RWMode            | ❌       |
| SegmentSize            | int64             | ❌       |
| NodeNum                | int64             | ❌       |
| SyncEnable             | bool              | ❌       |
| MaxFdNumsInCache       | int               | ❌       |
| CleanFdsCacheThreshold | float64           | ❌       |
| BufferSizeOfRecovery   | int               | ❌       |
| GCWhenClose            | bool              | ❌       |
| CommitBufferSize       | int64             | ❌       |
| ErrorHandler           | ErrorHandler      | ❌       |
| LessFunc               | LessFunc          | ❌       |
| MergeInterval          | Duration          | ❌       |
| MaxBatchCount          | int64             | ❌       |
| MaxBatchSize           | int64             | ❌       |
| ExpiredDeleteType      | ExpiredDeleteType | ❌       |
| MaxWriteRecordCount    | int64             | ❌       |
{{< /table >}}
