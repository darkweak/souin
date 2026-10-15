package otter

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/storages/core"
	"github.com/maypok86/otter"
	"github.com/pierrec/lz4/v4"
)

// Otter provider type.
type Otter struct {
	cache       *otter.StringBytesCache
	stale       time.Duration
	logger      core.Logger
	instanceKey int
}

var instanceMap = sync.Map{}

// Factory function create new Otter instance.
func Factory(otterCfg core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error) {
	defaultStorageSize := 10_000
	otterConfiguration := otterCfg.Configuration

	if otterConfiguration != nil {
		if oc, ok := otterConfiguration.(map[string]interface{}); ok {
			if v, found := oc["size"]; found && v != nil {
				val, _ := strconv.Atoi(fmt.Sprint(v))
				if val > 0 {
					defaultStorageSize = val
				} else if val, ok := v.(string); ok {
					intVal, _ := strconv.Atoi(val)
					if intVal > 0 {
						defaultStorageSize = intVal
					}
				}
			}
		}
	}

	if instance, ok := instanceMap.Load(defaultStorageSize); ok && instance != nil {
		cache := instance.(otter.StringBytesCache)

		return &Otter{
			cache:       &cache,
			stale:       stale,
			logger:      logger,
			instanceKey: defaultStorageSize,
		}, nil
	}

	cache, err := otter.NewStringBytesCache(defaultStorageSize)
	if err != nil {
		logger.Error("Impossible to instantiate the Otter DB.", err)
	}

	instanceMap.Store(defaultStorageSize, cache)
	logger.Infof("otter.storage.size %d", defaultStorageSize)

	return &Otter{cache: &cache, logger: logger, stale: stale, instanceKey: defaultStorageSize}, nil
}

// Name returns the storer name.
func (provider *Otter) Name() string {
	return "OTTER"
}

// Uuid returns an unique identifier.
func (provider *Otter) Uuid() string {
	return fmt.Sprint(provider.stale)
}

// MapKeys method returns a map with the key and value.
func (provider *Otter) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}

	provider.cache.Range(func(key string, val []byte) bool {
		if strings.HasPrefix(key, prefix) {
			k, _ := strings.CutPrefix(key, prefix)
			keys[k] = string(val)
		}

		return true
	})

	return keys
}

// ListKeys method returns the list of existing keys.
func (provider *Otter) ListKeys() []string {
	keys := []string{}

	provider.cache.Range(func(key string, value []byte) bool {
		if strings.HasPrefix(key, core.MappingKeyPrefix) {
			mapping, err := core.DecodeMapping(value)
			if err == nil {
				for _, v := range mapping.GetMapping() {
					keys = append(keys, v.GetRealKey())
				}
			}
		}

		return true
	})

	return keys
}

// Get method returns the populated response if exists, empty response then.
func (provider *Otter) Get(key string) []byte {
	result, found := provider.cache.Get(key)
	if !found {
		provider.logger.Debugf("Impossible to get the key %s in Otter", key)
	}

	return result
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Otter) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	val, found := provider.cache.Get(core.MappingKeyPrefix + key)
	if !found {
		provider.logger.Debugf("Impossible to get the mapping key %s in Otter", core.MappingKeyPrefix+key)

		return
	}

	fresh, stale, _ = core.MappingElection(provider, val, req, validator, provider.logger)

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Otter) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	compressed := new(bytes.Buffer)
	writer := lz4.NewWriter(compressed)

	if _, err := writer.Write(value); err != nil {
		_ = writer.Close()

		provider.logger.Errorf("Impossible to compress the key %s into Otter, %v", variedKey, err)

		return err
	}

	if err := writer.Close(); err != nil {
		provider.logger.Errorf("Impossible to close the compressor for key %s into Otter, %v", variedKey, err)

		return err
	}

	inserted := provider.cache.Set(variedKey, compressed.Bytes(), duration)
	if !inserted {
		provider.logger.Errorf("Impossible to set value into Otter, too large for the cost function")

		return nil
	}

	mappingKey := core.MappingKeyPrefix + baseKey
	item, _ := provider.cache.Get(mappingKey)

	val, e := core.MappingUpdater(variedKey, item, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	provider.logger.Debugf("Store the new mapping for the key %s in Otter", variedKey)
	// Used to calculate -(now * 2)
	negativeNow, err := time.ParseDuration(fmt.Sprintf("-%ds", time.Now().Nanosecond()*2))
	if err != nil {
		return fmt.Errorf("impossible to generate the duration: %w", err)
	}

	inserted = provider.cache.Set(mappingKey, val, negativeNow)
	if !inserted {
		provider.logger.Errorf("Impossible to set value into Otter, too large for the cost function")

		return nil
	}

	return nil
}

// Set method will store the response in Otter provider.
func (provider *Otter) Set(key string, value []byte, duration time.Duration) error {
	inserted := provider.cache.Set(key, value, duration)
	if !inserted {
		provider.logger.Errorf("Impossible to set value into Otter, too large for the cost function")
	}

	return nil
}

// Delete method will delete the response in Otter provider if exists corresponding to key param.
func (provider *Otter) Delete(key string) {
	provider.cache.Delete(key)
}

// DeleteMany method will delete the responses in Otter provider if exists corresponding to the regex key param.
func (provider *Otter) DeleteMany(key string) {
	rgKey, e := regexp.Compile(key)
	if e != nil {
		return
	}

	provider.cache.DeleteByFunc(func(k string, value []byte) bool {
		return rgKey.MatchString(k)
	})
}

// Init method will.
func (provider *Otter) Init() error {
	return nil
}

// Reset method will reset or close provider.
func (provider *Otter) Reset() error {
	provider.cache.Clear()

	// Only delete this instance from the cache
	instanceMap.Delete(provider.instanceKey)

	return nil
}
