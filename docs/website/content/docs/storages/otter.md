+++
weight = 406
title = "Otter"
icon = "home_storage"
description = "Otter is an in-memory/filesystem storage system"
tags = ["Beginners"]
+++

## What is Otter
Otter is a high performance lockless cache for Go. Many times faster than Ristretto and friends.

## Github repository
[https://github.com/maypok86/otter](https://github.com/maypok86/otter)

## Configuration
You can find the configuration for Otter with the values table below.  
By default the cache size is for 10_000 elements.

### Values
{{< table "table-hover" >}}
| Key name | type | required |
|----------|------|----------|
| size     | int  | ❌       |
{{< /table >}}
