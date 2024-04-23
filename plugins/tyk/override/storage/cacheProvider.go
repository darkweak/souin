package storage

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Cache provider type
type Cache struct {
	*cache.Cache
	logger *zap.Logger
	stale  time.Duration
}

// CacheConnectionFactory function create new Cache instance
func CacheConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	provider := cache.New(1*time.Second, 1*time.Second)
	return &Cache{
		Cache:  provider,
		logger: c.GetLogger(),
		stale:  c.GetDefaultCache().GetStale(),
	}, nil
}

// Name returns the storer name
func (provider *Cache) Name() string {
	return "CACHE"
}

// ListKeys method returns the list of existing keys
func (provider *Cache) ListKeys() []string {
	items := provider.Items()
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}

	return keys
}

// MapKeys method returns the map of existing keys
func (provider *Cache) MapKeys(prefix string) map[string]string {
	var keys map[string]string
	items := provider.Items()
	for k, v := range items {
		keys[k] = v.Object.(string)
	}

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
func (provider *Cache) Prefix(key string) []string {
	result := []string{}
	for k := range provider.Items() {
		if strings.HasPrefix(k, key) {
			result = append(result, k)
		}
	}

	return result
}

// Set method will store the response in Cache provider
func (provider *Cache) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	if duration == 0 {
		duration = url.TTL.Duration
	}

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

	for k := range provider.Items() {
		if re.MatchString(k) {
			provider.Delete(k)
		}
	}
}

// Init method will
func (provider *Cache) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Cache) Reset() error {
	provider.Cache.Flush()

	return nil
}

func (provider *Cache) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	r, found := provider.Cache.Get(MappingKeyPrefix + key)
	if !found {
		return
	}

	v, ok := r.([]byte)
	if !ok {
		return
	}

	if len(v) > 0 {
		fresh, stale, _ = mappingElection(provider, v, req, validator, provider.logger)
	}

	return fresh, stale
}
func (provider *Cache) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	var e error
	provider.Cache.Set(variedKey, value, duration+provider.stale)
	mappingKey := MappingKeyPrefix + baseKey
	r, found := provider.Cache.Get(mappingKey)
	if !found {
		return errors.New("key not found")
	}

	val, ok := r.([]byte)
	if !ok {
		return errors.New("value is not a byte slice")
	}

	val, e = mappingUpdater(variedKey, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
	if e != nil {
		return e
	}

	provider.logger.Sugar().Debugf("Store the new mapping for the key %s in storage", variedKey)
	provider.Cache.Set(mappingKey, val, -1)

	return nil
}
