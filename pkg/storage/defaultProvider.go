package storage

import (
	"bytes"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/storages/core"
	"github.com/pierrec/lz4/v4"
)

// Default provider type
type Default struct {
	m      *sync.Map
	stale  time.Duration
	logger core.Logger
}

type item struct {
	invalidAt time.Time
	value     []byte
}

// Factory function create new Default instance
func Factory(c configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
	return &Default{m: &sync.Map{}, logger: c.GetLogger(), stale: c.GetDefaultCache().GetStale()}, nil
}

// Name returns the storer name
func (provider *Default) Name() string {
	return types.DefaultStorageName
}

// Uuid returns an unique identifier
func (provider *Default) Uuid() string {
	return ""
}

// MapKeys method returns a map with the key and value
func (provider *Default) MapKeys(prefix string) map[string]string {
	now := time.Now()
	keys := map[string]string{}

	provider.m.Range(func(key, value any) bool {
		if strings.HasPrefix(key.(string), prefix) {
			k, _ := strings.CutPrefix(key.(string), prefix)
			if v, ok := value.(item); ok {
				if v.invalidAt.After(now) {
					keys[k] = string(v.value)
				}

				return true
			}
			if v, ok := value.(core.StorageMapper); ok {
				for _, v := range v.Mapping {
					if v.StaleTime.After(now) {
						keys[v.RealKey] = string(provider.Get(v.RealKey))
					}
				}
			}
		}

		return true
	})

	return keys
}

// ListKeys method returns the list of existing keys
func (provider *Default) ListKeys() []string {
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
func (provider *Default) Get(key string) []byte {
	result, ok := provider.m.Load(key)
	if !ok || result == nil {
		return nil
	}

	res, ok := result.(item)
	if !ok {
		return nil
	}

	if res.invalidAt.After(time.Now()) {
		return res.value
	}

	return nil
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Default) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	result, found := provider.m.Load(core.MappingKeyPrefix + key)
	if !found {
		return
	}

	fresh, stale, _ = core.MappingElection(provider, result.([]byte), req, validator, provider.logger)

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Default) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	var e error
	compressed := new(bytes.Buffer)
	if _, e = lz4.NewWriter(compressed).ReadFrom(bytes.NewReader(value)); e != nil {
		provider.logger.Errorf("Impossible to compress the key %s into Badger, %v", variedKey, e)
		return e
	}

	provider.m.Store(variedKey, item{
		invalidAt: now.Add(duration + provider.stale),
		value:     compressed.Bytes(),
	})

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

	provider.logger.Debugf("Store the new mapping for the key %s in Default", variedKey)
	provider.m.Store(mappingKey, val)
	return nil
}

// Set method will store the response in Badger provider
func (provider *Default) Set(key string, value []byte, duration time.Duration) error {
	provider.m.Store(key, item{
		invalidAt: time.Now().Add(duration),
		value:     value,
	})

	return nil
}

// Delete method will delete the response in Badger provider if exists corresponding to key param
func (provider *Default) Delete(key string) {
	provider.m.Delete(key)
}

// DeleteMany method will delete the responses in Badger provider if exists corresponding to the regex key param
func (provider *Default) DeleteMany(key string) {
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
func (provider *Default) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Default) Reset() error {
	provider.m = &sync.Map{}
	return nil
}
