+++
weight = 1
draft = true
title = "Introduction"
icon = "extension"
toc = true
description = "Resources to use Souin as a middleware in your favorite reverse-proxy"
tags = ["Beginners", "Advanced"]
+++


## What is Souin?

Souin is a powerful HTTP cache system written in go and implements the following RFCs (Request For Comments):
* [RFC-2616](https://datatracker.ietf.org/doc/html/rfc2616)
* [RFC-7234](https://datatracker.ietf.org/doc/html/rfc7234)
* [RFC-9111](https://datatracker.ietf.org/doc/html/rfc9111)
* [RFC-9211](https://datatracker.ietf.org/doc/html/rfc9211)
* [Cache-Groups (draft)](https://datatracker.ietf.org/doc/draft-ietf-httpbis-cache-groups/)
* [HTTP Cache invalidation (draft)](https://datatracker.ietf.org/doc/draft-nottingham-http-invalidation/)


## Multiple backend storages
### Local in-memory or filesystem
* [Badger](/docs/storages/badger)
* [Nuts](/docs/storages/nuts)

### Distributed in-memory
* [Redis](/docs/storages/redis)
* [Etcd](/docs/storages/etcd)
* [Olric](/docs/storages/olric)


## Fully customizable
The default configuration format is YAML (using the standalone HTTP cache server) because it's programmatically generable and easier than the VCL but unlike the VCL it's not a script language.
