+++
weight = 502
title = "Caddy"
icon = "extension"
description = "Use Souin directly in the Caddy web server"
tags = ["Beginners", "Advanced"]
+++

## Disclaimer
{{% alert icon=" " %}}
`github.com/darkweak/souin/plugin/caddy` and `github.com/caddyserver/cache-handler` are mainly the same but the `souin/plugin/caddy` is the development repository and the `cache-handler` is the stable version. They both contain the features and suport the configuration below but on the Souin repository you'll get access to new features/RFCs at early stage with faster bug fixes.
{{% /alert %}}

## Usage

### Build your caddy binary
We assume that you already installed the `xcaddy` binary on your device. If not, you can refer to the [documentation here](https://github.com/caddyserver/xcaddy#install)

```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy
```

You should get a new `caddy` executable file in your current directory.


### Run your HTTP cache
We need to tell caddy that it must use the HTTP cache with the `cache` global and handler directives. You can set your configuration globally and override it in each handlers.
```caddyfile
{
    debug
    cache {
        ttl 1h
    }
}

route {
    cache {
        ttl 30s
    }
    respond "Hello HTTP cache"
}
```

## Configuration
Every configuration directives can be used either in the global or in the handler blocks.

### Basic configuration
Usually we set the `ttl`, the `stale` and the `default_cache_control` directives in the global configuration.

```caddyfile
{
    ...
    cache {
        ttl 100s
        stale 3h
        default_cache_control public, s-maxage=100
    }
}
```

But we can go further by enabling the Souin API and enable the `debug`, `prometheus`, `souin` endpoints
```caddyfile
{
    ...
    cache {
        api {
            debug
            prometheus
            souin
        }
    }
}

route {
    cache
}
```
With this given configuration if you go on [https://localhost/souin-api/souin](https://localhost/souin-api/souin) we get the stored keys list.  
If we go on [https://localhost/souin-api/metrics](https://localhost/souin-api/metrics) we access to the prometheus web page.  
If we go on [https://localhost/souin-api/debug/](https://localhost/souin-api/debug/) we access to the pprof web page.  

### Complex configuration

#### Storages
{{% alert context="warning" %}}
Since `v1.7.0` Souin implements only an in-memory storage, if you need a specific storage you have to take it from [the storages repository](https://github.com/darkweak/storages) and add to your build command.  
(e.g. with otter using caddy) You have to build your caddy module with the desired storage 
```shell
xcaddy build --with github.com/darkweak/souin/plugins/caddy --with github.com/darkweak/storages/otter/caddy
```
and configure otter in your Caddyfile/JSON configuration file.  
See the [storages page]({{% relref "/docs/storages" %}}) to learn more about each supported storage.
{{% /alert %}}

First you have to build Caddy with Souin and a storage using the following template.
```
xcaddy build \
    --with github.com/darkweak/souin/plugins/caddy \
    --with github.com/darkweak/storages/{your_storage_name}/caddy
```

You can also use as many storages you want.
```
xcaddy build \
    --with github.com/darkweak/souin/plugins/caddy \
    --with github.com/darkweak/storages/redis/caddy \
    --with github.com/darkweak/storages/nuts/caddy \
    --with github.com/darkweak/storages/otter/caddy
```

We can define multiple storages to use to store the response from the upstream server and specify the order.
Here, we define 3 storages `badger`, `nuts` and `redis` and `nuts` will be accessed first, `badger` the second and `redis` the third only if the previous doesn't return suitable data.

```caddyfile
{
    ...
    cache {
        badger {
            path /tmp/badger/default
        }
        nuts {
            path /tmp/nuts/default
        }
        redis {
            url 127.0.0.1:6379
        }

        storers nuts badger redis
    }
}
```

Indeed you can configure each storage with the `path` or `url` directive (depending the provider) but also with the `configuration` directive. That's a simple `key - value` that are defined by the library used to implement each provider.
```caddyfile
route /nuts-path {
    cache {
        nuts {
            path /tmp/nuts/file
        }
    }
}

route /nuts-configuration {
    cache {
        nuts {
            configuration {
                Dir /tmp/nuts-configuration
                EntryIdxMode 1
                RWMode 0
                SegmentSize 1024
                NodeNum 42
                SyncEnable true
                StartFileLoadingMode 1
            }
        }
    }
    respond "Hello nuts"
}
```
