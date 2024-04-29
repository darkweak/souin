package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/dgraph-io/ristretto"
	"github.com/imdario/mergo"
	"github.com/nutsdb/nutsdb"
	lz4 "github.com/pierrec/lz4/v4"
	"go.uber.org/zap"
)

var nutsMemcachedInstanceMap = map[string]*nutsdb.DB{}

// Why NutsMemcached?
// ---
// The NutsMemcached storage backend is composed of two different storage backends:
// 1. NutsDB: for the cache key index (i.e., IDX_ keys).
// 2. Memcached: for the cache content.
// There are two storage backends because:
// 1. is a "non forgetting" storage backend (NutsDB, for the index). Keys will be kept until their TTL expires.
//  → if it was handled by a storage backend that can preemptively evict, you might evict IDX_ keys, which you wouldn't want.
//    You need to make sure index and content stays in sync.
// 2. is "forgetting" storage backend (Memcached, for the data). Cache data will be pre-emptively evicted (i.e., before TTL is reached).
//  → it makes it possible to put limits on total RAM/disk usage.

// NutsMemcached provider type
type NutsMemcached struct {
	*nutsdb.DB
	stale  time.Duration
	logger *zap.Logger
	//memcacheClient *memcache.Client
	ristrettoCache *ristretto.Cache
}

// Below is already defined in the original Nuts provider.
/* const (
	bucket    = "souin-bucket"
	nutsLimit = 1 << 16
)

func sanitizeProperties(m map[string]interface{}) map[string]interface{} {
	iotas := []string{"RWMode", "StartFileLoadingMode"}
	for _, i := range iotas {
		if v := m[i]; v != nil {
			currentMode := nutsdb.FileIO
			switch v {
			case 1:
				currentMode = nutsdb.MMap
			}
			m[i] = currentMode
		}
	}

	for _, i := range []string{"SegmentSize", "NodeNum", "MaxFdNumsInCache"} {
		if v := m[i]; v != nil {
			m[i], _ = v.(int64)
		}
	}

	if v := m["EntryIdxMode"]; v != nil {
		m["EntryIdxMode"] = nutsdb.HintKeyValAndRAMIdxMode
		switch v {
		case 1:
			m["EntryIdxMode"] = nutsdb.HintKeyAndRAMIdxMode
		}
	}

	if v := m["SyncEnable"]; v != nil {
		m["SyncEnable"] = true
		if b, ok := v.(bool); ok {
			m["SyncEnable"] = b
		} else if s, ok := v.(string); ok {
			m["SyncEnable"], _ = strconv.ParseBool(s)
		}
	}

	return m
} */

// NutsConnectionFactory function create new NutsMemcached instance
func NutsMemcachedConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	dc := c.GetDefaultCache()
	nutsConfiguration := dc.GetNutsMemcached()
	nutsOptions := nutsdb.DefaultOptions
	nutsOptions.Dir = "/tmp/souin-nuts-memcached"

	// `HintKeyAndRAMIdxMode` represents ram index (only key) mode.
	nutsOptions.EntryIdxMode = nutsdb.HintKeyAndRAMIdxMode
	// `HintBPTSparseIdxMode` represents b+ tree sparse index mode.
	// Note: this mode was removed after v0.14.0
	// Use: github.com/nutsdb/nutsdb v0.14.0
	//nutsOptions.EntryIdxMode = nutsdb.HintBPTSparseIdxMode

	// EntryIdxMode will affect the size of the key index in memory.
	// → since this storage backend has no limit on memory usage, it has to be chosen depending on
	//   the max number of cache keys that will be kept in flight.

	if nutsConfiguration.Configuration != nil {
		var parsedNuts nutsdb.Options
		nutsConfiguration.Configuration = sanitizeProperties(nutsConfiguration.Configuration.(map[string]interface{}))
		if b, e := json.Marshal(nutsConfiguration.Configuration); e == nil {
			if e = json.Unmarshal(b, &parsedNuts); e != nil {
				c.GetLogger().Sugar().Error("Impossible to parse the configuration for the Nuts provider", e)
			}
		}

		if err := mergo.Merge(&nutsOptions, parsedNuts, mergo.WithOverride); err != nil {
			c.GetLogger().Sugar().Error("An error occurred during the nutsOptions merge from the default options with your configuration.")
		}
	} else {
		nutsOptions.RWMode = nutsdb.MMap
		if nutsConfiguration.Path != "" {
			nutsOptions.Dir = nutsConfiguration.Path
		}
	}

	if instance, ok := nutsMemcachedInstanceMap[nutsOptions.Dir]; ok && instance != nil {
		return &NutsMemcached{
			DB:     instance,
			stale:  dc.GetStale(),
			logger: c.GetLogger(),
		}, nil
	}

	db, e := nutsdb.Open(nutsOptions)

	if e != nil {
		c.GetLogger().Sugar().Error("Impossible to open the Nuts DB.", e)
		return nil, e
	}

	// Ristretto config
	var numCounters int64 = 1e7 // number of keys to track frequency of (10M).
	var maxCost int64 = 1 << 30 // maximum cost of cache (1GB).
	if nutsConfiguration.Configuration != nil {
		rawNumCounters, ok := nutsConfiguration.Configuration.(map[string]interface{})["NumCounters"]
		if ok {
			numCounters, _ = strconv.ParseInt(rawNumCounters.(string), 10, 64)
		}

		rawMaxCost, ok := nutsConfiguration.Configuration.(map[string]interface{})["MaxCost"]
		if ok {
			maxCost, _ = strconv.ParseInt(rawMaxCost.(string), 10, 64)
		}
	}
	// See https://github.com/dgraph-io/ristretto?tab=readme-ov-file#example
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: numCounters,
		MaxCost:     maxCost,
		BufferItems: 64, // number of keys per Get buffer.
	})
	if err != nil {
		c.GetLogger().Sugar().Error("Impossible to make new Ristretto cache.", err)
		return nil, e
	}

	instance := &NutsMemcached{
		DB:     db,
		stale:  dc.GetStale(),
		logger: c.GetLogger(),
		//memcacheClient: memcache.New("127.0.0.1:11211"), // hardcoded for now
		ristrettoCache: ristrettoCache,
	}
	nutsMemcachedInstanceMap[nutsOptions.Dir] = instance.DB

	return instance, nil
}

// Name returns the storer name
func (provider *NutsMemcached) Name() string {
	return "NUTS_MEMCACHED"
}

// ListKeys method returns the list of existing keys
func (provider *NutsMemcached) ListKeys() []string {
	keys := []string{}

	e := provider.DB.View(func(tx *nutsdb.Tx) error {
		e, _ := tx.PrefixScan(bucket, []byte(MappingKeyPrefix), 0, 100)
		for _, k := range e {
			mapping, err := decodeMapping(k.Value)
			if err == nil {
				for _, v := range mapping.Mapping {
					keys = append(keys, v.RealKey)
				}
			}
		}
		return nil
	})

	if e != nil {
		return []string{}
	}

	return keys
}

// MapKeys method returns the map of existing keys
func (provider *NutsMemcached) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}

	e := provider.DB.View(func(tx *nutsdb.Tx) error {
		e, _ := tx.GetAll(bucket)
		for _, k := range e {
			if strings.HasPrefix(string(k.Key), prefix) {
				nk, _ := strings.CutPrefix(string(k.Key), prefix)
				keys[nk] = string(k.Value)
			}
		}
		return nil
	})

	if e != nil {
		return map[string]string{}
	}

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *NutsMemcached) Get(key string) (item []byte) {
	memcachedKey, _ := provider.getFromNuts(key)

	if memcachedKey != "" {
		item, _ = provider.getFromMemcached(memcachedKey)
	}

	return
}

// Prefix method returns the populated response if exists, empty response then
func (provider *NutsMemcached) Prefix(key string) []string {
	result := []string{}

	_ = provider.DB.View(func(tx *nutsdb.Tx) error {
		prefix := []byte(key)

		if entries, err := tx.PrefixSearchScan(bucket, prefix, "^({|$)", 0, 50); err != nil {
			return err
		} else {
			for _, entry := range entries {
				result = append(result, string(entry.Key))
			}
		}
		return nil
	})

	return result
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *NutsMemcached) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	_ = provider.DB.View(func(tx *nutsdb.Tx) error {
		i, e := tx.Get(bucket, []byte(MappingKeyPrefix+key))
		if e != nil && !errors.Is(e, nutsdb.ErrKeyNotFound) {
			return e
		}

		var val []byte
		if i != nil {
			val = i.Value
		}
		fresh, stale, e = mappingElection(provider, val, req, validator, provider.logger)

		return e
	})

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *NutsMemcached) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	compressed := new(bytes.Buffer)
	if _, err := lz4.NewWriter(compressed).ReadFrom(bytes.NewReader(value)); err != nil {
		provider.logger.Sugar().Errorf("Impossible to compress the key %s into Nuts, %v", variedKey, err)
		return err
	}
	{
		// matchedURL is only use when ttl == 0
		ttl := duration + provider.stale
		url := t.URL{
			TTL: configurationtypes.Duration{Duration: ttl},
		}
		err := provider.Set(variedKey, compressed.Bytes(), url, ttl)
		if err != nil {
			return err
		}
	}

	err := provider.DB.Update(func(tx *nutsdb.Tx) error {
		mappingKey := MappingKeyPrefix + baseKey
		item, e := tx.Get(bucket, []byte(mappingKey))
		if e != nil && !errors.Is(e, nutsdb.ErrKeyNotFound) {
			provider.logger.Sugar().Errorf("Impossible to get the base key %s in Nuts, %v", baseKey, e)
			return e
		}

		var val []byte
		if item != nil {
			val = item.Value
		}

		val, e = mappingUpdater(variedKey, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
		if e != nil {
			return e
		}

		provider.logger.Sugar().Debugf("Store the new mapping for the key %s in Nuts", variedKey)

		return tx.Put(bucket, []byte(mappingKey), val, nutsdb.Persistent)
	})

	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Nuts, %v", err)
	}

	return err
}

// Set method will store the response in Nuts provider
func (provider *NutsMemcached) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	if duration == 0 {
		duration = url.TTL.Duration
	}
	// Only for memcached (to overcome 250 bytes key limit)
	//memcachedKey := uuid.New().String()
	// Disabled for ristretto to improve performances
	memcachedKey := key

	// set to nuts
	{
		err := provider.DB.Update(func(tx *nutsdb.Tx) error {
			// key: cache-key, value: memcached-key
			return tx.Put(bucket, []byte(key), []byte(memcachedKey), uint32(duration.Seconds()))
		})

		if err != nil {
			provider.logger.Sugar().Errorf("Impossible to set value into Nuts, %v", err)
			return err
		}
	}

	// set to memcached
	_ = provider.setToMemcached(memcachedKey, value, int32(duration.Seconds()))
	return nil
}

// Delete method will delete the response in Nuts provider if exists corresponding to key param
func (provider *NutsMemcached) Delete(key string) {
	memcachedKey, _ := provider.getFromNuts(key)

	// delete from memcached
	if memcachedKey != "" {
		_ = provider.delFromMemcached(memcachedKey)
	}

	// delete from nuts
	_ = provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.Delete(bucket, []byte(key))
	})
}

// DeleteMany method will delete the responses in Nuts provider if exists corresponding to the regex key param
func (provider *NutsMemcached) DeleteMany(keyReg string) {
	_ = provider.DB.Update(func(tx *nutsdb.Tx) error {
		if entries, err := tx.PrefixSearchScan(bucket, []byte(""), keyReg, 0, nutsLimit); err != nil {
			return err
		} else {
			for _, entry := range entries {
				// delete from memcached
				_ = provider.delFromMemcached(string(entry.Value))
				// delete from nuts
				_ = tx.Delete(bucket, entry.Key)
			}
		}
		return nil
	})
}

// Init method will
func (provider *NutsMemcached) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *NutsMemcached) Reset() error {
	return provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.DeleteBucket(1, bucket)
	})
}

func (provider *NutsMemcached) getFromNuts(nutsKey string) (memcachedKey string, err error) {
	err = provider.DB.View(func(tx *nutsdb.Tx) error {
		i, e := tx.Get(bucket, []byte(nutsKey))
		if i != nil {
			memcachedKey = string(i.Value)
		}
		return e
	})
	return
}

// Reminder: the memcachedKey must be at most 250 bytes in length
func (provider *NutsMemcached) setToMemcached(memcachedKey string, value []byte, ttl int32) (err error) {
	//fmt.Println("memcached SET", key)
	// err = provider.memcacheClient.Set(
	// 	&memcache.Item{
	// 		Key:        memcachedKey,
	// 		Value:      value,
	// 		Expiration: ttl,
	// 	},
	// )
	//if err != nil {
	// 	provider.logger.Sugar().Errorf("Failed to set into memcached, %v", err)
	// }
	ok := provider.ristrettoCache.Set(memcachedKey, value, int64(len(value)))
	if !ok {
		provider.logger.Sugar().Debugf("Value not set to ristretto cache, key=%v", memcachedKey)
	}
	return
}

// Reminder: the memcachedKey must be at most 250 bytes in length
func (provider *NutsMemcached) getFromMemcached(memcachedKey string) (value []byte, err error) {
	//fmt.Println("memcached GET", key)
	// i, err := provider.memcacheClient.Get(memcachedKey)
	// if err == nil && i != nil {
	// 	value = i.Value
	// } else {
	// 	provider.logger.Sugar().Errorf("Failed to get from memcached, %v", err)
	// }
	rawValue, found := provider.ristrettoCache.Get(memcachedKey)
	if !found {
		provider.logger.Sugar().Debugf("Failed to get from cache, key=%v", memcachedKey)
		return nil, errors.New("failed to get from cache")
	}
	value = rawValue.([]byte)
	return
}

func (provider *NutsMemcached) delFromMemcached(memcachedKey string) (err error) {
	//err = provider.memcacheClient.Delete(memcachedKey)
	provider.ristrettoCache.Del(memcachedKey)
	return
}
