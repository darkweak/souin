+++
weight = 405
title = "Olric"
icon = "home_storage"
description = "Olric is a distributed in-memory storage system"
tags = ["Beginners"]
+++

## What is Olric
{{% alert %}}
The Olric instance must connect to an external service (olric service) that you run on your own.
{{% /alert %}}
Olric is a distributed, in-memory object store. It's designed from the ground up to be distributed, and it can be used both as an embedded Go library and as a language-independent service.

With Olric, you can instantly create a fast, scalable, shared pool of RAM across a cluster of computers.

Olric is implemented in Go and uses the Redis serialization protocol. So Olric has client implementations in all major programming languages.

Olric is highly scalable and available. Distributed applications can use it for distributed caching, clustering and publish-subscribe messaging.

## Github repository
[https://github.com/buraksezer/olric](https://github.com/buraksezer/olric)

## Configuration
{{% alert context="warning" %}}
You can't configure in Souin the Olric server instance.
{{% /alert %}}

### Values
{{% alert context="info" %}}
There are no values to configure Olric because it just connects to the external Olric server.
{{% /alert %}}
