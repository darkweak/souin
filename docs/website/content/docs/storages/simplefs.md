+++
weight = 410
title = "Simplefs"
icon = "home_storage"
description = "Simplefs is a filesystem storage system"
tags = ["Beginners"]
+++

## What is Simplefs
Simplefs is a high performance filesystem cache for Go. It was built because no other simple filesystem existed.

## Github repository
[https://github.com/darkweak/storages](https://github.com/darkweak/storages)

## Use Otter
### With Caddy
You have to build your caddy instance including `Souin` and `Simplefs` using `xcaddy` ([refer to the build caddy section]({{% relref "/docs/middlewares/caddy#build-your-caddy-binary" %}})).
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/simplefs/caddy
```
You will be able to use otter in your Caddyfile or JSON configuration file.
```caddyfile
{
    cache {
        ttl 1h
        simplefs {
            configuration {
                size 10000
                path /somewhere/to/store
                directory_size 100MB
            }
        }
    }
}

route {
    cache
    respond "Hello HTTP cache"
}
```

## Configuration
You can find the configuration for Simplefs with the values table below.  

### Values
{{< table "table-hover" >}}
| Key name       |  type  | required |
|----------------|--------|----------|
| size           | int    | ✅       |
| path           | string | ❌       |
| directory_size | int    | ❌       |
{{< /table >}}
