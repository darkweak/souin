package storage

import (
	"bufio"
	"bytes"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/akyoto/cache"
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
)

// Cache provider type
type Cache struct {
	*cache.Cache
	stale time.Duration
}

var sharedCache *Cache

// Factory function create new Cache instance
func Factory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	provider := cache.New(1 * time.Second)

	if sharedCache == nil {
		sharedCache = &Cache{Cache: provider, stale: c.GetDefaultCache().GetStale()}
	}

	return sharedCache, nil
}

// Name returns the storer name
func (provider *Cache) Name() string {
	return "CACHE"
}

// Uuid returns an unique identifier
func (provider *Cache) Uuid() string {
	return ""
}

// ListKeys method returns the list of existing keys
func (provider *Cache) ListKeys() []string {
	var keys []string
	provider.Cache.Range(func(key, _ interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})

	return keys
}

// MapKeys method returns the map of existing keys
func (provider *Cache) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}
	provider.Cache.Range(func(key, value interface{}) bool {
		if strings.HasPrefix(key.(string), prefix) {
			k, _ := strings.CutPrefix(key.(string), prefix)
			keys[k] = string(value.([]byte))
		}
		return true
	})

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Cache) Get(key string) []byte {
	result, found := provider.Cache.Get(key)

	if !found {
		return []byte{}
	}

	return result.([]byte)
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Cache) GetMultiLevel(key string, req *http.Request, validator *types.Revalidator) (fresh *http.Response, stale *http.Response) {
	result, found := provider.Cache.Get("IDX_" + key)
	if !found {
		return
	}

	fresh, stale, _ = rfc.MappingElection(provider, result.([]byte), req, validator)

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Cache) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	var e error

	provider.Cache.Set(variedKey, value, duration)

	mappingKey := "IDX_" + baseKey
	item, ok := provider.Cache.Get(mappingKey)
	var val []byte
	if ok {
		val = item.([]byte)
	}

	val, e = rfc.MappingUpdater(variedKey, val, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	provider.Cache.Set(mappingKey, val, 0)
	return nil
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Cache) Prefix(key string, req *http.Request, validator *types.Revalidator) *http.Response {
	var result *http.Response

	provider.Cache.Range(func(k, v interface{}) bool {
		if !strings.HasPrefix(k.(string), key) {
			return true
		}

		if k == key || varyVoter(key, req, k.(string)) {
			if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(v.([]byte))), req); err == nil {
				rfc.ValidateETagFromHeader(res.Header.Get("etag"), validator)
				if validator.Matched {
					result = res
					return false
				}
			}
			return true
		}

		return true
	})

	return result
}

// Set method will store the response in Cache provider
func (provider *Cache) Set(key string, value []byte, duration time.Duration) error {
	provider.Cache.Set(key, value, duration)

	return nil
}

// Delete method will delete the response in Cache provider if exists corresponding to key param
func (provider *Cache) Delete(key string) {
	provider.Cache.Delete(key)
}

// DeleteMany method will delete the responses in Cache provider if exists corresponding to the regex key param
func (provider *Cache) DeleteMany(key string) {
	re, e := regexp.Compile(key)

	if e != nil {
		return
	}

	provider.Cache.Range(func(k, _ interface{}) bool {
		if re.MatchString(k.(string)) {
			provider.Delete(k.(string))
		}
		return true
	})
}

// Init method will
func (provider *Cache) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Cache) Reset() error {
	provider.DeleteMany("*")

	return nil
}
