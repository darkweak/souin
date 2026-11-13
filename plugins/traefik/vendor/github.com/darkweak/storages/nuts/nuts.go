package nuts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"dario.cat/mergo"
	"github.com/darkweak/storages/core"
	"github.com/nutsdb/nutsdb"
)

var nutsInstanceMap = sync.Map{}

// Nuts provider type.
type Nuts struct {
	*nutsdb.DB

	stale       time.Duration
	logger      core.Logger
	uuid        string
	instanceKey string
	bucketReady atomic.Bool
}

const (
	bucket    = "souin-bucket"
	nutsLimit = 1 << 16
)

func sanitizeProperties(configMap map[string]interface{}) map[string]interface{} {
	for _, iteration := range []string{"RWMode", "StartFileLoadingMode"} {
		if v := configMap[iteration]; v != nil {
			switch v {
			case nutsdb.FileIO, fmt.Sprint(nutsdb.FileIO), "FileIO":
				configMap[iteration] = nutsdb.FileIO
			case nutsdb.MMap, fmt.Sprint(nutsdb.MMap), "MMap":
				configMap[iteration] = nutsdb.MMap
			}
		}
	}

	// int types
	for _, iteration := range []string{"MaxFdNumsInCache", "BufferSizeOfRecovery"} {
		if v := configMap[iteration]; v != nil {
			if val, ok := v.(int); ok {
				configMap[iteration] = val
			}

			if val, ok := v.(string); ok {
				configMap[iteration], _ = strconv.Atoi(val)
			}
		}
	}

	// int64 types
	for _, iteration := range []string{"SegmentSize", "NodeNum", "CleanFdsCacheThreshold", "CommitBufferSize", "MaxBatchCount", "MaxBatchSize", "MaxWriteRecordCount"} {
		if v := configMap[iteration]; v != nil {
			if val, ok := v.(int64); ok {
				configMap[iteration] = val
			}

			if val, ok := v.(string); ok {
				configMap[iteration], _ = strconv.ParseInt(val, 10, 64)
			}
		}
	}

	// time.Duration types
	for _, iteration := range []string{"MergeInterval"} {
		if v := configMap[iteration]; v != nil {
			if val, ok := v.(time.Duration); ok {
				configMap[iteration] = val
			}

			if val, ok := v.(string); ok {
				var err error
				if configMap["MergeInterval"], err = time.ParseDuration(val); err != nil {
					i, _ := strconv.Atoi(val)
					configMap["MergeInterval"] = time.Duration(i)
				}
			}
		}
	}

	// bool types
	for _, iteration := range []string{"SyncEnable", "GCWhenClose"} {
		if v := configMap[iteration]; v != nil {
			configMap[iteration] = true
			if b, ok := v.(bool); ok {
				configMap[iteration] = b
			} else if s, ok := v.(string); ok {
				configMap[iteration], _ = strconv.ParseBool(s)
			}
		}
	}

	// custom types
	if v := configMap["EntryIdxMode"]; v != nil {
		switch v {
		case nutsdb.HintKeyAndRAMIdxMode, fmt.Sprint(nutsdb.HintKeyAndRAMIdxMode), "HintKeyAndRAMIdxMode", "KeyAndRAMIdxMode":
			configMap["EntryIdxMode"] = nutsdb.HintKeyAndRAMIdxMode
		case nutsdb.HintKeyValAndRAMIdxMode, fmt.Sprint(nutsdb.HintKeyValAndRAMIdxMode), "HintKeyValAndRAMIdxMode", "KeyValAndRAMIdxMode":
			configMap["EntryIdxMode"] = nutsdb.HintKeyValAndRAMIdxMode
		}
	}

	if v := configMap["ExpiredDeleteType"]; v != nil {
		switch v {
		case nutsdb.TimeWheel, fmt.Sprint(nutsdb.TimeWheel), "TimeWheel":
			configMap["ExpiredDeleteType"] = nutsdb.TimeWheel
		case nutsdb.TimeHeap, fmt.Sprint(nutsdb.TimeHeap), "TimeHeap":
			configMap["ExpiredDeleteType"] = nutsdb.TimeHeap
		}
	}

	return configMap
}

// Factory function create new Nuts instance.
func Factory(nutsConfiguration core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error) {
	nutsOptions := nutsdb.DefaultOptions
	nutsOptions.Dir = "/tmp/souin-nuts"

	if nutsConfiguration.Configuration != nil {
		var parsedNuts nutsdb.Options

		nutsConfiguration.Configuration = sanitizeProperties(nutsConfiguration.Configuration.(map[string]interface{}))
		if b, e := json.Marshal(nutsConfiguration.Configuration); e == nil {
			if e = json.Unmarshal(b, &parsedNuts); e != nil {
				logger.Error("Impossible to parse the configuration for the Nuts provider", e)
			}
		}

		if err := mergo.Merge(&nutsOptions, parsedNuts, mergo.WithOverride); err != nil {
			logger.Error("An error occurred during the nutsOptions merge from the default options with your configuration.")
		}
	} else {
		nutsOptions.RWMode = nutsdb.MMap
		if nutsConfiguration.Path != "" {
			nutsOptions.Dir = nutsConfiguration.Path
		}
	}

	if instance, ok := nutsInstanceMap.Load(nutsOptions.Dir); ok && instance != nil {
		return &Nuts{
			DB:     instance.(*nutsdb.DB),
			stale:  stale,
			logger: logger,
		}, nil
	}

	database, err := nutsdb.Open(nutsOptions)
	if err != nil {
		logger.Error("Impossible to open the Nuts DB.", err)

		if errors.Is(err, nutsdb.ErrCrc) {
			_ = os.Remove(nutsOptions.Dir)

			return Factory(nutsConfiguration, logger, stale)
		}

		if errors.Is(err, nutsdb.ErrDirLocked) {
			// Retry once after one second, the db should be present in the sync map
			time.Sleep(time.Second)

			if instance, ok := nutsInstanceMap.Load(nutsOptions.Dir); ok && instance != nil {
				return &Nuts{
					DB:     instance.(*nutsdb.DB),
					stale:  stale,
					logger: logger,
				}, nil
			} else {
				return nil, err
			}
		}

		return nil, err
	}

	instance := &Nuts{
		DB:          database,
		stale:       stale,
		logger:      logger,
		uuid:        fmt.Sprintf("%s-%s", nutsOptions.Dir, stale),
		instanceKey: nutsOptions.Dir,
	}
	nutsInstanceMap.Store(nutsOptions.Dir, instance.DB)

	return instance, nil
}

// Name returns the storer name.
func (provider *Nuts) Name() string {
	return "NUTS"
}

// Uuid returns an unique identifier.
func (provider *Nuts) Uuid() string {
	return provider.uuid
}

// ListKeys method returns the list of existing keys.
func (provider *Nuts) ListKeys() []string {
	keys := []string{}

	err := provider.View(func(tx *nutsdb.Tx) error {
		values, _ := tx.PrefixScan(bucket, []byte(core.MappingKeyPrefix), 0, 100)
		for _, v := range values {
			mapping, err := core.DecodeMapping(v)
			if err == nil {
				for _, v := range mapping.GetMapping() {
					keys = append(keys, v.GetRealKey())
				}
			}
		}

		return nil
	})
	if err != nil {
		return []string{}
	}

	return keys
}

// MapKeys method returns the map of existing keys.
func (provider *Nuts) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}
	bytePrefix := []byte(prefix)

	err := provider.View(func(tx *nutsdb.Tx) error {
		nKeys, values, _ := tx.GetAll(bucket)
		for iteration, v := range values {
			k := nKeys[iteration]
			if bytes.HasPrefix(k, bytePrefix) {
				nk, _ := strings.CutPrefix(string(k), prefix)
				keys[nk] = string(v)
			}
		}

		return nil
	})
	if err != nil {
		return map[string]string{}
	}

	return keys
}

// Get method returns the populated response if exists, empty response then.
func (provider *Nuts) Get(key string) []byte {
	var item []byte

	_ = provider.View(func(tx *nutsdb.Tx) error {
		v, e := tx.Get(bucket, []byte(key))
		if v != nil {
			item = v
		}

		return e
	})

	return item
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Nuts) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	_ = provider.View(func(tx *nutsdb.Tx) error {
		value, err := tx.Get(bucket, []byte(core.MappingKeyPrefix+key))
		if err != nil && !errors.Is(err, nutsdb.ErrKeyNotFound) {
			return err
		}

		var val []byte
		if value != nil {
			val = value
		}

		fresh, stale, err = core.MappingElection(provider, val, req, validator, provider.logger)

		return err
	})

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Nuts) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	compressed, err := core.CompressValue(value)
	if err != nil {
		provider.logger.Errorf("Impossible to compress the key %s into Nuts, %v", variedKey, err)

		return err
	}

	provider.ensureBucket()

	// Store the response and its mapping in a single transaction: one commit,
	// hence one sync, instead of one per write.
	err = provider.Update(func(ntx *nutsdb.Tx) error {
		if e := ntx.Put(bucket, []byte(variedKey), compressed, uint32((duration + provider.stale).Seconds())); e != nil {
			provider.logger.Errorf("Impossible to set the key %s into Nuts, %v", variedKey, e)

			return e
		}

		mappingKey := core.MappingKeyPrefix + baseKey
		item, err := ntx.Get(bucket, []byte(mappingKey))

		if err != nil && !errors.Is(err, nutsdb.ErrKeyNotFound) {
			provider.logger.Errorf("Impossible to get the base key %s in Nuts, %v", baseKey, err)

			return err
		}

		var val []byte
		if item != nil {
			val = item
		}

		val, err = core.MappingUpdater(variedKey, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
		if err != nil {
			return err
		}

		provider.logger.Debugf("Store the new mapping for the key %s in Nuts", variedKey)

		return ntx.Put(bucket, []byte(mappingKey), val, nutsdb.Persistent)
	})
	if err != nil {
		provider.logger.Errorf("Impossible to set value into Nuts, %v", err)
	}

	return err
}

// Set method will store the response in Nuts provider.
func (provider *Nuts) Set(key string, value []byte, duration time.Duration) error {
	provider.ensureBucket()

	err := provider.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucket, []byte(key), value, uint32(duration.Seconds()))
	})
	if err != nil {
		provider.logger.Errorf("Impossible to set value into Nuts, %v", err)
	}

	return err
}

// Delete method will delete the response in Nuts provider if exists corresponding to key param.
func (provider *Nuts) Delete(key string) {
	_ = provider.Update(func(tx *nutsdb.Tx) error {
		return tx.Delete(bucket, []byte(key))
	})
}

// DeleteMany method will delete the responses in Nuts provider if exists corresponding to the regex key param.
func (provider *Nuts) DeleteMany(key string) {
	rgKey, err := regexp.Compile(key)
	if err != nil {
		provider.logger.Errorf("The key %s is not a valid regexp: %v", key, err)

		return
	}

	_ = provider.Update(func(ntx *nutsdb.Tx) error {
		if entries, err := ntx.GetKeys(bucket); err != nil {
			return err
		} else {
			for _, entry := range entries {
				if rgKey.Match(entry) {
					_ = ntx.Delete(bucket, entry)
				}
			}
		}

		return nil
	})
}

// Init method will.
func (provider *Nuts) Init() error {
	return nil
}

// Reset method will reset or close provider.
func (provider *Nuts) Reset() error {
	var err error
	// Close the DB connection
	if provider.DB != nil {
		err = provider.Close()
	}
	// Only delete this instance from the cache
	nutsInstanceMap.Delete(provider.instanceKey)

	return err
}

// ensureBucket creates the bucket the first time this instance writes. Every
// write used to try creating it in a dedicated transaction, which fails once
// the bucket exists but still costs a write transaction each time.
func (provider *Nuts) ensureBucket() {
	if provider.bucketReady.Load() {
		return
	}

	err := provider.Update(func(tx *nutsdb.Tx) error {
		return tx.NewBucket(nutsdb.DataStructureBTree, bucket)
	})
	if err == nil || errors.Is(err, nutsdb.ErrBucketAlreadyExist) {
		provider.bucketReady.Store(true)
	}
}
