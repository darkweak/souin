+++
weight = 516
title = "Tyk"
icon = "extension"
description = "Use Souin directly in the Tyk web server"
tags = ["Beginners", "Advanced"]
+++

## Compile the Souin extension for tyk
You can compile your own Souin integration using the `Makefile` and the `docker-compose` inside the [tyk integration directory](https://github.com/darkweak/souin/tree/master/plugins/tyk) to generate the `souin-plugin.so` file.


## Usage
To use Souin as Træfik plugin, you have to define the use of Souin as `post` and `response` custom middleware. Place your previously generated `souin-plugin.so` file inside your `middleware` directory.
```json
{
  "name":"httpbin.org",
  "api_id":"3",
  "org_id":"3",
  "use_keyless": true,
  "version_data": {
    "not_versioned": true,
    "versions": {
      "Default": {
        "name": "Default",
        "use_extended_paths": true
      }
    }
  },
  "custom_middleware": {
    "pre": [],
    "post": [
      {
        "name": "SouinRequestHandler",
        "path": "/opt/tyk-gateway/middleware/souin-plugin.so"
      }
    ],
    "post_key_auth": [],
    "auth_check": {
      "name": "",
      "path": "",
      "require_session": false
    },
    "response": [
      {
        "name": "SouinResponseHandler",
        "path": "/opt/tyk-gateway/middleware/souin-plugin.so"
      }
    ],
    "driver": "goplugin",
    "id_extractor": {
      "extract_from": "",
      "extract_with": "",
      "extractor_config": {}
    }
  },
  "proxy":{
    "listen_path":"/httpbin/",
    "target_url":"http://httpbin.org/",
    "strip_listen_path":true
  },
  "active":true,
  "config_data": {
    "httpcache": {
      "default_cache": {
        "default_cache_control": "public, max-age:=3600",
        "ttl": "5s",
        "stale": "1d"
      }
    }
  }
}
```
With that your application will be able to cache the responses if possible and returns at least the `Cache-Status` HTTP header with the different directives mentionned in the RFC specification.

Look at the configuration section to discover [all configurable keys here]({{% relref "/docs/configuration" %}}).

Other resources
---------------
You can find an example for a docker-compose stack inside the [`tyk` folder on the Github repository](https://github.com/darkweak/souin/tree/master/plugins/tyk).
