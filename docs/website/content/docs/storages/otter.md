+++
weight = 408
title = "Otter"
icon = "home_storage"
description = "Otter is an in-memory/filesystem storage system"
tags = ["Beginners"]
+++

## What is Otter
Otter is a high performance lockless cache for Go. Many times faster than Ristretto and friends.

## Github repository
[https://github.com/maypok86/otter](https://github.com/maypok86/otter)

## Use Otter
### With Caddy
You have to build your caddy instance including `Souin` and `Otter` using `xcaddy` ([refer to the build caddy section]({{% relref "/docs/middlewares/caddy#build-your-caddy-binary" %}})).
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/otter/caddy
```
You will be able to use otter in your Caddyfile or JSON configuration file.
```caddyfile
{
    cache {
        ttl 1h
        otter
    }
}

route {
    cache
    respond "Hello HTTP cache"
}
```

## Configuration
You can find the configuration for Otter with the values table below.  
By default the cache size is for 10_000 elements.

### Values
{{< table "table-hover" >}}
| Key name | type | required |
|----------|------|----------|
| size     | int  | ❌       |
{{< /table >}}
