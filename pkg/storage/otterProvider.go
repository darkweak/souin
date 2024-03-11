package storage

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/maypok86/otter"
	"go.uber.org/zap"
)

// Otter provider type
type Otter struct {
	cache  *otter.CacheWithVariableTTL[string, []byte]
	stale  time.Duration
	logger *zap.Logger
}

// OtterConnectionFactory function create new Otter instance
func OtterConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	cache, e := otter.MustBuilder[string, []byte](10_000).
		CollectStats().
		Cost(func(key string, value []byte) uint32 {
			return 1
		}).
		WithVariableTTL().
		Build()

	if e != nil {
		c.GetLogger().Sugar().Error("Impossible to instanciate the Otter DB.", e)
	}

	dc := c.GetDefaultCache()

	return &Otter{cache: &cache, logger: c.GetLogger(), stale: dc.GetStale()}, nil
}

// Name returns the storer name
func (provider *Otter) Name() string {
	return "OTTER"
}

// MapKeys method returns a map with the key and value
func (provider *Otter) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}

	provider.cache.Range(func(key string, val []byte) bool {
		if !strings.HasPrefix(key, prefix) {
			k, _ := strings.CutPrefix(string(key), prefix)
			keys[k] = string(val)
		}

		return true
	})

	return keys
}

// ListKeys method returns the list of existing keys
func (provider *Otter) ListKeys() []string {
	keys := []string{}

	provider.cache.Range(func(key string, _ []byte) bool {
		if !strings.Contains(key, surrogatePrefix) {
			keys = append(keys, key)
		}

		return true
	})

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Otter) Get(key string) []byte {
	result, found := provider.cache.Get(key)
	if !found {
		provider.logger.Sugar().Errorf("Impossible to get the key %s in Otter", key)
	}

	return result
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Otter) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	var result *http.Response
	provider.cache.Range(func(k string, val []byte) bool {
		if varyVoter(key, req, k) {
			if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(val)), req); err == nil {
				rfc.ValidateETag(res, validator)
				if validator.Matched {
					provider.logger.Sugar().Debugf("The stored key %s matched the current iteration key ETag %+v", k, validator)
					result = res
					return false
				} else {
					provider.logger.Sugar().Debugf("The stored key %s didn't match the current iteration key ETag %+v", k, validator)
				}
			} else {
				provider.logger.Sugar().Errorf("An error occured while reading response for the key %s: %v", k, err)
			}
		}

		return true
	})

	return result
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Otter) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	val, found := provider.cache.Get(MappingKeyPrefix + key)
	if !found {
		provider.logger.Sugar().Errorf("Impossible to get the mapping key %s in Otter", MappingKeyPrefix+key)
		return
	}

	fresh, stale, _ = mappingElection(provider, val, req, validator, provider.logger)
	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Otter) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration) error {
	now := time.Now()
	inserted := provider.cache.Set(variedKey, value, duration)
	if !inserted {
		provider.logger.Sugar().Errorf("Impossible to set value into Otter, too large for the cost function")
		return nil
	}

	mappingKey := MappingKeyPrefix + baseKey
	item, found := provider.cache.Get(mappingKey)
	if !found {
		provider.logger.Sugar().Errorf("Impossible to get the base key %s in Otter", mappingKey)
		return nil
	}

	val, e := mappingUpdater(variedKey, item, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag)
	if e != nil {
		return e
	}

	provider.logger.Sugar().Debugf("Store the new mapping for the key %s in Badger, %v", variedKey, string(val))
	// Used to calculate -(now * 2)
	negativeNow, _ := time.ParseDuration(fmt.Sprintf("-%d", time.Now().Nanosecond()*2))
	inserted = provider.cache.Set(mappingKey, val, negativeNow)
	if !inserted {
		provider.logger.Sugar().Errorf("Impossible to set value into Otter, too large for the cost function")
		return nil
	}

	return nil
}

// Set method will store the response in Badger provider
func (provider *Otter) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	inserted := provider.cache.Set(key, value, duration)
	if !inserted {
		provider.logger.Sugar().Errorf("Impossible to set value into Otter, too large for the cost function")
	}

	return nil
}

// Delete method will delete the response in Badger provider if exists corresponding to key param
func (provider *Otter) Delete(key string) {
	provider.cache.Delete(key)
}

// DeleteMany method will delete the responses in Badger provider if exists corresponding to the regex key param
func (provider *Otter) DeleteMany(key string) {
	re, e := regexp.Compile(key)
	if e != nil {
		return
	}

	provider.cache.DeleteByFunc(func(k string, value []byte) bool {
		return re.MatchString(k)
	})
}

// Init method will
func (provider *Otter) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Otter) Reset() error {
	provider.cache.Clear()

	return nil
}
