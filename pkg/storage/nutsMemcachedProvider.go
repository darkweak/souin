package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/dgraph-io/ristretto"
	"github.com/imdario/mergo"
	"github.com/nutsdb/nutsdb"
	"go.uber.org/zap"
)

var nutsMemcachedInstanceMap = map[string]*nutsdb.DB{}

// NutsMemcached provider type
type NutsMemcached struct {
	*nutsdb.DB
	stale  time.Duration
	logger *zap.Logger
	//memcacheClient *memcache.Client
	ristrettoCache *ristretto.Cache
}

// const (
// 	bucket    = "souin-bucket"
// 	nutsLimit = 1 << 16
// )

// func sanitizeProperties(m map[string]interface{}) map[string]interface{} {
// 	iotas := []string{"RWMode", "StartFileLoadingMode"}
// 	for _, i := range iotas {
// 		if v := m[i]; v != nil {
// 			currentMode := nutsdb.FileIO
// 			switch v {
// 			case 1:
// 				currentMode = nutsdb.MMap
// 			}
// 			m[i] = currentMode
// 		}
// 	}

// 	for _, i := range []string{"SegmentSize", "NodeNum", "MaxFdNumsInCache"} {
// 		if v := m[i]; v != nil {
// 			m[i], _ = v.(int64)
// 		}
// 	}

// 	if v := m["EntryIdxMode"]; v != nil {
// 		m["EntryIdxMode"] = nutsdb.HintKeyValAndRAMIdxMode
// 		switch v {
// 		case 1:
// 			m["EntryIdxMode"] = nutsdb.HintKeyAndRAMIdxMode
// 		}
// 	}

// 	if v := m["SyncEnable"]; v != nil {
// 		m["SyncEnable"] = true
// 		if b, ok := v.(bool); ok {
// 			m["SyncEnable"] = b
// 		} else if s, ok := v.(string); ok {
// 			m["SyncEnable"], _ = strconv.ParseBool(s)
// 		}
// 	}

// 	return m
// }

// NutsConnectionFactory function create new Nuts instance
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
		panic(err)
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
		e, _ := tx.GetAll(bucket)
		for _, k := range e {
			if !strings.Contains(string(k.Key), surrogatePrefix) {
				keys = append(keys, string(k.Key))
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
func (provider *NutsMemcached) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	var result *http.Response

	_ = provider.DB.View(func(tx *nutsdb.Tx) error {
		prefix := []byte(key)

		if entries, err := tx.PrefixSearchScan(bucket, prefix, "^({|$)", 0, 50); err != nil {
			return err
		} else {
			for _, entry := range entries {
				if varyVoter(key, req, string(entry.Key)) {
					// TODO: improve this
					// Store only response header in nuts and avoid query to memcached on each vary
					// E.g, rfc.ValidateETag on NutsDB header value, retrieve response body later from memcached.

					// Reminder: the key must be at most 250 bytes in length
					//fmt.Println("memcached PREFIX", key, "GET", string(entry.Key))
					i, e := provider.getFromMemcached(string(entry.Value))
					if e == nil {
						res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(i)), req)
						if err == nil {
							rfc.ValidateETag(res, validator)
							if validator.Matched {
								provider.logger.Sugar().Debugf("The stored key %s matched the current iteration key ETag %+v", string(entry.Key), validator)
								result = res
								return nil
							}

							provider.logger.Sugar().Debugf("The stored key %s didn't match the current iteration key ETag %+v", string(entry.Key), validator)
						} else {
							provider.logger.Sugar().Errorf("An error occured while reading response for the key %s: %v", string(entry.Key), err)
						}
					} else {
						provider.logger.Sugar().Errorf("An error occured while reading memcached for the key %s: %v", string(entry.Key), err)
					}
				}
			}
		}
		return nil
	})

	return result
}

// Set method will store the response in Nuts provider
func (provider *NutsMemcached) Set(key string, value []byte, url t.URL, ttl time.Duration) error {
	if ttl == 0 {
		ttl = url.TTL.Duration
	}
	// Only for memcached (to overcome 250 bytes key limit)
	//memcachedKey := uuid.New().String()
	memcachedKey := key

	// set to nuts (normal TTL)
	{
		err := provider.DB.Update(func(tx *nutsdb.Tx) error {

			// key: cache-key, value: memcached-key
			return tx.Put(bucket, []byte(key), []byte(memcachedKey), uint32(ttl.Seconds()))
		})

		if err != nil {
			provider.logger.Sugar().Errorf("Impossible to set value into Nuts, %v", err)
			return err
		}
	}

	// set to nuts (stale TTL)
	staleTtl := int32((provider.stale + ttl).Seconds())
	{
		err := provider.DB.Update(func(tx *nutsdb.Tx) error {
			// key: "STALE_" + cache-key, value: memcached-key
			return tx.Put(bucket, []byte(StalePrefix+key), []byte(memcachedKey), uint32(staleTtl))
		})

		if err != nil {
			provider.logger.Sugar().Errorf("Impossible to set value into Nuts, %v", err)
		}
	}

	// set to memcached with stale TTL
	_ = provider.setToMemcached(memcachedKey, value, staleTtl)

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
		provider.logger.Sugar().Debugf("Value not set to cache, key=%v", memcachedKey)
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
