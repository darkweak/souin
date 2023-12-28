+++
weight = 401
title = "Badger"
icon = "home_storage"
description = "Badger is an in-memory storage system"
tags = ["Beginners"]
+++

## What is Badger
BadgerDB is an embeddable, persistent and fast key-value (KV) database written in pure Go. It is the underlying database for Dgraph, a fast, distributed graph database. It's meant to be a performant alternative to non-Go-based key-value stores like RocksDB.

## Github repository
[https://github.com/dgraph-io/badger](https://github.com/dgraph-io/badger)

## Configuration
You can find the configuration for Badger [here](https://github.com/dgraph-io/badger/blob/main/options.go#L44) or check the values table below.

### Values
{{< table "table-hover" >}}
| Key name                      | type          | required |
|-------------------------------|---------------|----------|
| Dir                           | string        | ✅       |
| ValueDir                      | string        | ✅       |
| SyncWrites                    | bool          | ❌       |
| NumVersionsToKeep             | int           | ❌       |
| ReadOnly                      | bool          | ❌       |
| InMemory                      | bool          | ❌       |
| MetricsEnabled                | bool          | ❌       |
| NumGoroutines                 | int           | ❌       |
| MemTableSize                  | int64         | ❌       |
| BaseTableSize                 | int64         | ❌       |
| BaseLevelSize                 | int64         | ❌       |
| LevelSizeMultiplier           | int           | ❌       |
| TableSizeMultiplier           | int           | ❌       |
| MaxLevels                     | int           | ❌       |
| VLogPercentile                | float64       | ❌       |
| ValueThreshold                | int64         | ❌       |
| NumMemtables                  | int           | ❌       |
| BlockSize                     | int           | ❌       |
| BloomFalsePositive            | float64       | ❌       |
| BlockCacheSize                | int64         | ❌       |
| IndexCacheSize                | int64         | ❌       |
| NumLevelZeroTables            | int           | ❌       |
| NumLevelZeroTablesStall       | int           | ❌       |
| ValueLogFileSize              | int64         | ❌       |
| ValueLogMaxEntries            | uint32        | ❌       |
| NumCompactors                 | int           | ❌       |
| CompactL0OnClose              | bool          | ❌       |
| LmaxCompaction                | bool          | ❌       |
| ZSTDCompressionLevel          | int           | ❌       |
| VerifyValueChecksum           | bool          | ❌       |
| EncryptionKey                 | []byte        | ❌       |
| EncryptionKeyRotationDuration | time.Duration | ❌       |
| BypassLockGuard               | bool          | ❌       |
| DetectConflicts               | bool          | ❌       |
| NamespaceOffset               | int           | ❌       |
| ExternalMagicVersion          | uint16        | ❌       |
{{< /table >}}
