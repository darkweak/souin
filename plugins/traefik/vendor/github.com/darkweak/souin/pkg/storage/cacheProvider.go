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

// CacheConnectionFactory function create new Cache instance
func CacheConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	provider := cache.New(1 * time.Second)
	return &Cache{Cache: provider, stale: c.GetDefaultCache().GetStale()}, nil
}

// Name returns the storer name
func (provider *Cache) Name() string {
	return "CACHE"
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
	var keys map[string]string
	provider.Cache.Range(func(key, value interface{}) bool {
		k, _ := strings.CutPrefix(key.(string), prefix)
		keys[k] = value.(string)
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

// Prefix method returns the populated response if exists, empty response then
func (provider *Cache) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	var result *http.Response

	provider.Cache.Range(func(k, v interface{}) bool {
		if !strings.HasPrefix(k.(string), key) {
			return true
		}

		if k == key || varyVoter(key, req, k.(string)) {
			if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(v.([]byte))), req); err == nil {
				rfc.ValidateETag(res, validator)
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
func (provider *Cache) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	provider.Cache.Set(key, value, duration)
	provider.Cache.Set(StalePrefix+key, value, provider.stale+duration)

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
