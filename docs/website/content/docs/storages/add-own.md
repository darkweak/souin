+++
weight = 410
title = "Add your own storage"
icon = "home_storage"
description = "Badger is an in-memory storage system"
tags = ["Beginners"]
+++

## Context
The [storage repository](https://github.com/darkweak/storages) defines an interface called `Storer` in the [core/core.go file](https://github.com/darkweak/storages/blob/main/core/core.go) that should be implemented by your storage.  
Your `struct` must implement it to be a valid storer and be registered in the storage pool.

By convention we declare a `Factory` function that respects this signature:
```go
func(providerConfiguration core.CacheProvider, logger *zap.Logger, stale time.Duration) (core.Storer, error)
```

And the `Storer` interface is the following:
```go
type Storer interface {
	MapKeys(prefix string) map[string]string
	ListKeys() []string
	Get(key string) []byte
	Set(key string, value []byte, duration time.Duration) error
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Name() string
	Uuid() string
	Reset() error

	// Multi level storer to handle fresh/stale at once
	GetMultiLevel(key string, req *http.Request, validator *Revalidator) (fresh *http.Response, stale *http.Response)
	SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error
}
```

## Example
Let's define our simple in-memory storage
```go
// your_custom_storage.go
package your_package

import (
	"sync"
	"time"

	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/storages/core"
	"go.uber.org/zap"
)

// custom storage provider type
type customStorage struct {
	m      *sync.Map
	stale  time.Duration
	logger *zap.Logger
}

// Factory function create new custom storage instance
func Factory(_ core.CacheProvider, logger *zap.Logger, stale time.Duration) (types.Storer, error) {
	return &customStorage{m: &sync.Map{}, logger: logger, stale: stale}, nil
}
```

We have to implement the `Storer` interface now.
```go
// your_custom_storage.go
...

// Name returns the storer name
func (provider *customStorage) Name() string {
	return "YOUR_CUSTOM_STORAGE"
}

// Uuid returns an unique identifier
func (provider *customStorage) Uuid() string {
	return "THE_UUID"
}

// MapKeys method returns a map with the key and value
func (provider *customStorage) MapKeys(prefix string) map[string]string {
	now := time.Now()
	keys := map[string]string{}

	provider.m.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), prefix) {
			k, _ := strings.CutPrefix(key.(string), prefix)
			if v, ok := value.(core.StorageMapper); ok {
				for _, v := range v.Mapping {
					if v.StaleTime.After(now) {
						keys[v.RealKey] = string(provider.Get(v.RealKey))
					}
				}

				return true
			}

			keys[k] = string(value.([]byte))
		}

		return true
	})

	return keys
}

// ListKeys method returns the list of existing keys
func (provider *customStorage) ListKeys() []string {
	now := time.Now()
	keys := []string{}

	provider.m.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), core.MappingKeyPrefix) {
			mapping, err := core.DecodeMapping(value.([]byte))
			if err == nil {
				for _, v := range mapping.Mapping {
					if v.StaleTime.After(now) {
						keys = append(keys, v.RealKey)
					} else {
						provider.m.Delete(v.RealKey)
					}
				}
			}
		}

		return true
	})

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *customStorage) Get(key string) []byte {
	result, ok := provider.m.Load(key)
	if !ok || result == nil {
		return nil
	}

	res, ok := result.([]byte)
	if !ok {
		return nil
	}

	return res
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *customStorage) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	result, found := provider.m.Load(core.MappingKeyPrefix + key)
	if !found {
		return
	}

	fresh, stale, _ = core.MappingElection(provider, result.([]byte), req, validator, provider.logger)

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *customStorage) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	var e error
	compressed := new(bytes.Buffer)
	if _, e = lz4.NewWriter(compressed).ReadFrom(bytes.NewReader(value)); e != nil {
		provider.logger.Sugar().Errorf("Impossible to compress the key %s into Badger, %v", variedKey, e)
		return e
	}

	provider.m.Store(variedKey, compressed.Bytes())

	mappingKey := core.MappingKeyPrefix + baseKey
	item, ok := provider.m.Load(mappingKey)
	var val []byte
	if ok {
		val = item.([]byte)
	}

	val, e = core.MappingUpdater(variedKey, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	provider.logger.Sugar().Debugf("Store the new mapping for the key %s in customStorage", variedKey)
	provider.m.Store(mappingKey, val)
	return nil
}

// Set method will store the response in Badger provider
func (provider *customStorage) Set(key string, value []byte, duration time.Duration) error {
	provider.m.Store(key, value)

	return nil
}

// Delete method will delete the response in Badger provider if exists corresponding to key param
func (provider *customStorage) Delete(key string) {
	provider.m.Delete(key)
}

// DeleteMany method will delete the responses in Badger provider if exists corresponding to the regex key param
func (provider *customStorage) DeleteMany(key string) {
	re, e := regexp.Compile(key)

	if e != nil {
		return
	}

	provider.m.Range(func(key, _ any) bool {
		if re.MatchString(key.(string)) {
			provider.m.Delete(key)
		}

		return true
	})
}

// Init method will
func (provider *customStorage) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *customStorage) Reset() error {
	provider.m = &sync.Map{}
	return nil
}
```

After that you'll be able to register your storage using
```go
// anywhere.go
...
logger, _ := zap.NewProduction()
customStorage, _ := your_package.Factory(core.CacheProvider{}, logger, time.Hour)
// It will register as `YOUR_CUSTOM_STORAGE-THE_UUID`.
core.RegisterStorage(customStorage)

// In your code
if st := core.GetRegisteredStorer("YOUR_CUSTOM_STORAGE-THE_UUID"); st != nil {
    customStorage = st.(types.Storer)
}
```
