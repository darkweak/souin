+++
weight = 300
title = "Configuration"
icon = "settings"
description = "Discover how to configure Souin properly"
tags = ["Beginners", "Advanced", "in-memory"]
+++

## Keys details

### API
The api prefix configures the APIs exposed by Souin. (e.g. `api.basepath`)
* **basepath**: Basepath for all APIs to avoid conflicts (e.g. `/your-non-conflicting-route`)  
default: `/souin-api`


#### Pprof API
The debug prefix configures the internal pprof API. (e.g. `api.debug.enable`)
* **basepath**: Basepath for the pprof API to avoid conflicts (e.g. `/another-pprof-api-path`)  
default: `/debug`, full url would be `/souin-api/debug`

* **enable**: Enable the internal pprof API.  
default: `false`

#### Prometheus API
The prometheus prefix configures the internal prometheus API. (e.g. `api.prometheus.enable`)
* **basepath**: Basepath for the prometheus API to avoid conflicts (e.g. `/another-prometheus-api-path`)  
default: `/metrics`, full url would be `/souin-api/metrics`

* **enable**: Enable the internal prometheus API.  
default: `false`

#### Souin API
The souin prefix configures the internal souin API. (e.g. `api.souin.enable`)
* **basepath**: Basepath for the souin API to avoid conflicts (e.g. `/another-souin-api-path`)  
default: `/souin`, full url would be `/souin-api/souin`


### Cache keys
The cache_keys prefix configures the key generation rules for each URI matching the key. (e.g. `cache_keys..*\.css.disable_body`)

{{% alert icon=" " %}}
For a YAML configuration it should look like that
```yaml
cache_keys:
    .*\.css:
        disable_body: true
        disable_host: true
        disable_method: true
        disable_query: true
        headers:
            - Authorization
            - Content-Type
```
{{% /alert %}}

* **disable_body**: Prevent the body hash to be part of the generated key.  
default: `false`

* **disable_host**: Prevent the hostname to be part of the generated key.  
default: `false`

* **disable_method**: Prevent the HTTP method to be part of the generated key.  
default: `false`

* **disable_query**: Prevent the URL query to be part of the generated key.  
default: `false`

* **disable_scheme**: Prevent the scheme to be part of the generated key.  
default: `false`

* **hash**: Hash the key in the storage.  
default: `false`

* **headers**: Add specific headers to the generated key.  

* **hide**: Prevent the key from being exposed in the `Cache-Status` HTTP response header.  
default: `false`


### CDN
The cdn prefix configure the upfront CDN yuo have placed before Souin. It can handle ofr you the cache invalidation on your CDN, the optimization to have your pages cached directly on it.

* **provider**: The provider name placed before Souin.  
**values**: akamai, cloudflare, fastly, souin

* **dynamic**: Enable the dynamic keys returned by your backend application and store them in the surrogate storage even if they are not declared in your `surrogate-keys` configuration.  
default: `false`

* **api_key**: The api key used to access to the provider if required, depending the provider.  
[required by Cloudflare, Fastly]

* **email**: The email used to access to the provider if required, depending the provider.  
[required by Cloudflare]

* **hostname**: The hostname if required, depending the provider.  
[required by Akamai]

* **network**: The network if required, depending the provider.  
[required by Akamai]

* **strategy**: The strategy to use to purge the cdn cache, soft will keep the content as a stale resource.  
[required by Akamai, Fastly]

* **service_id**: The service id if required, depending the provider.  
[required by Fastly]

* **zone_id**: The zone id if required, depending the provider.  
[required by Cloudflare]


### Default cache
The default_cache prefix configure the default cache behavior. (e.g. `default_cache.allowed_http_verbs`).

#### Allowed HTTP verbs
The allowed_http_verbs prefix configure the HTTP verbs allowed to get cached. (e.g. `default_cache.allowed_http_verbs`).  
default: `[GET, HEAD]`

#### Badger
The badger prefix configure the badger storage. (e.g. `default_cache.badger`).

* **path**: Configure the Badger storage directory.

* **configuration**: Configure Badger directly in your configuration file.  
[See the Badger configuration for the options]({{% relref "/docs/storages/badger" %}})

#### Default Cache-Control
The default_cache_control prefix configure the Cache-Control to set if the upstream server doesn't return any. (e.g. `default_cache.default_cache_control`).  
example: `public, max-age=3600`

#### Distributed
The distributed prefix configure if the storage must use a distributed storage. (e.g. `default_cache.distributed`).  
default: `true`

#### Etcd
The etcd prefix configure the etcd storage. (e.g. `default_cache.etcd`).

* **configuration**: Configure Etcd directly in your configuration file.  
[See the Etcd configuration for the options]({{% relref "/docs/storages/etcd" %}})

#### Key
The key prefix override the key generation with the ability to disable unecessary parts. (e.g. `default_cache.key.disable_body`).

* **disable_body**: Prevent the body hash to be part of the generated key.  
default: `false`

* **disable_host**: Prevent the hostname to be part of the generated key.  
default: `false`

* **disable_method**: Prevent the HTTP method to be part of the generated key.  
default: `false`

* **disable_query**: Prevent the URL query to be part of the generated key.  
default: `false`

* **disable_scheme**: Prevent the scheme to be part of the generated key.  
default: `false`

* **hash**: Hash the key in the storage.  
default: `false`

* **headers**: Add specific headers to the generated key.  

* **hide**: Prevent the key from being exposed in the `Cache-Status` HTTP response header.  
default: `false`

#### Max cacheable body bytes
Limit to define if the body size is allowed to be cached. (e.g. `1048576` (1MB)).  
If a limit is set, your streamed/chunk responses won't be cached.  
default: `unlimited`

#### Mode
The mode prefix allow you to bypass some RFC requirements. (e.g. `default_cache.mode`).  
default: `strict`

The value can be one of these:
* **mode**: Prevent the body hash to be part of the generated key.  
* **bypass**: Fully bypass the RFC requirements.
* **bypass_request**: Bypass the request RFC requirements.
* **bypass_response**: Bypass the response RFC requirements.
* **strict**: Respect the RFC requirements.

#### Nuts
The nuts prefix configure the nuts storage. (e.g. `default_cache.nuts`).

* **path**: Configure the Nuts storage directory.

* **configuration**: Configure Nuts directly in your configuration file.  
[See the Nuts configuration for the options]({{% relref "/docs/storages/nuts" %}})

#### Olric
The olric prefix configure the olric storage. (e.g. `default_cache.olric`).

* **path**: Configure the Olric storage with a YAML file.

* **configuration**: Configure the Embedded Olric instance directly in your configuration file. It won't connect to an external olric instance.  
[See the Embedded Olric configuration for the options]({{% relref "/docs/storages/embedded-olric" %}})

#### Otter
The otter prefix configure the otter storage. (e.g. `default_cache.otter`).

* **configuration**: Configure Otter directly in your configuration file.  
[See the Otter configuration for the options]({{% relref "/docs/storages/otter" %}})

#### Regex
The regex prefix configure the actions to do on URL that match the regex. (e.g. `default_cache.regex`).

* **exclude**: The regex used to prevent paths being cached.  
example: `^[A-z]+.*$`

#### Stale
The stale prefix configure the duration to keep the stale responses in the storage. (e.g. `default_cache.stale`).  
example: `1d`

#### Storers
The storers prefix configure the order to use the storages, with that you'll be able to chain them, use a local in-memory and fallback to a redis or distributed one that is slower. (e.g. `default_cache.storers`).  
example: `[nuts otter badger]`

#### Timeout
The timeout prefix configure the timeouts. (e.g. `default_cache.timeout`).

* **backend**: Duration before considering the backend upstream as unreachable.
example: `10s`

* **cache**: Duration before considering the cache storages as unreachable.
example: `10ms`

#### TTL
The ttl prefix configure the duration to keep the fresh responses in the storage. (e.g. `default_cache.ttl`).  
example: `10m`


### Log level
The log_level prefix configure the log verbosity. (e.g. `log_level`).  
default: `INFO`

The value can be one of these:
* `DEBUG`
* `INFO`
* `WARN`
* `ERROR`
* `DPANIC`
* `PANIC`
* `FATAL`


### Reverse-proxy
The reverse_proxy_url prefix configure the reverse-proxy URL to contact. Only required using the standalone server. (e.g. `reverse_proxy_url`).  
example: `http://yourdomain.com:81`


### Surrogate-keys
The surrogate_keys prefix configures the surrogates keys to associate the response depending the matched header values or url. (e.g. `surrogate_keys.something_key_group./something`)  
The surrogate-keys is a way to group the cache keys together and invalidate one group at once instead of invalidating each keys separately. It's very efficient in combination to the [cdn configuration]({{% relref "/docs/configuration#cdn" %}}), with that souin will be able to group and invalidate your cached resources directly on your CDN.

{{% alert icon=" " %}}
For a YAML configuration it should look like that
```yaml
surrogate_keys:
    all_responses_with_content_type:
        headers:
            Content-Type: '.+'
        url: /my-path
```
{{% /alert %}}

* **headers**: Headers values to match to associate the response to this surrogate-key. It's a simple key-value, the key is the header name and the value is the regex to match.

* **url**: URL to match to associate the response to this surrogate-key.  
example: `/something/.+`


### URLs
The urls prefix configures the ttl or the default_cache_control value for each requests that match the given regex. (e.g. `url.https:\/\/yourdomain.com.ttl`)

{{% alert icon=" " %}}
For a YAML configuration it should look like that
```yaml
urls:
    https:\/\/yourdomain.com:
        ttl: 10s
        default_cache_control: public, max-age=86400
```
{{% /alert %}}

* **ttl**: Override the default TTL if defined.  
example: `10s`

* **default_cache_control**: Override the default default Cache-Control if defined.  
example: `public, max-age=86400`


### Ykeys
The ykeys prefix configures the ykeys to associate the response depending the matched header values or url. (e.g. `ykeys.something_key_group./something`)

{{% alert icon=" " %}}
For a YAML configuration it should look like that
```yaml
ykeys:
    all_responses_with_content_type:
        headers:
            Content-Type: '.+'
        url: /my-path
```
{{% /alert %}}

* **headers**: Headers values to match to associate the response to this ykey. It's a simple key-value, the key is the header name and the value is the regex to match.

* **url**: URL to match to associate the response to this ykey.  
example: `/something/.+`
