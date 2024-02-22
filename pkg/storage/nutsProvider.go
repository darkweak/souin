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
	"github.com/imdario/mergo"
	"github.com/nutsdb/nutsdb"
	"go.uber.org/zap"
)

var nutsInstanceMap = map[string]*nutsdb.DB{}

// Nuts provider type
type Nuts struct {
	*nutsdb.DB
	stale  time.Duration
	logger *zap.Logger
}

const (
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
}

// NutsConnectionFactory function create new Nuts instance
func NutsConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	dc := c.GetDefaultCache()
	nutsConfiguration := dc.GetNuts()
	nutsOptions := nutsdb.DefaultOptions
	nutsOptions.Dir = "/tmp/souin-nuts"
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

	if instance, ok := nutsInstanceMap[nutsOptions.Dir]; ok && instance != nil {
		return &Nuts{
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

	instance := &Nuts{
		DB:     db,
		stale:  dc.GetStale(),
		logger: c.GetLogger(),
	}
	nutsInstanceMap[nutsOptions.Dir] = instance.DB

	return instance, nil
}

// Name returns the storer name
func (provider *Nuts) Name() string {
	return "NUTS"
}

// ListKeys method returns the list of existing keys
func (provider *Nuts) ListKeys() []string {
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
func (provider *Nuts) MapKeys(prefix string) map[string]string {
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
func (provider *Nuts) Get(key string) (item []byte) {
	_ = provider.DB.View(func(tx *nutsdb.Tx) error {
		i, e := tx.Get(bucket, []byte(key))
		if i != nil {
			item = i.Value
		}
		return e
	})

	return
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Nuts) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	var result *http.Response

	_ = provider.DB.View(func(tx *nutsdb.Tx) error {
		prefix := []byte(key)

		if entries, err := tx.PrefixSearchScan(bucket, prefix, "^({|$)", 0, 50); err != nil {
			return err
		} else {
			for _, entry := range entries {
				if varyVoter(key, req, string(entry.Key)) {
					if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(entry.Value)), req); err == nil {
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
				}
			}
		}
		return nil
	})

	return result
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Nuts) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	_ = provider.DB.View(func(tx *nutsdb.Tx) error {
		i, e := tx.Get(bucket, []byte(mappingKeyPrefix+key))
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

// SetMultiLevel tries to store the keywith the given value and update the mapping key to store metadata.
func (provider *Nuts) SetMultiLevel(baseKey, key string, value []byte, variedHeaders http.Header, etag string, duration time.Duration) error {
	now := time.Now()

	err := provider.DB.Update(func(tx *nutsdb.Tx) error {
		e := tx.Put(bucket, []byte(key), value, uint32((duration + provider.stale).Seconds()))
		if e != nil {
			provider.logger.Sugar().Errorf("Impossible to set the key %s into Nuts, %v", key, e)
		}

		return e
	})

	if err != nil {
		return err
	}

	err = provider.DB.Update(func(tx *nutsdb.Tx) error {
		mappingKey := mappingKeyPrefix + baseKey
		item, e := tx.Get(bucket, []byte(mappingKey))
		if e != nil && !errors.Is(e, nutsdb.ErrKeyNotFound) {
			provider.logger.Sugar().Errorf("Impossible to get the base key %s in Nuts, %v", baseKey, e)
			return e
		}

		var val []byte
		if item != nil {
			val = item.Value
		}

		val, e = mappingUpdater(key, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag)
		if e != nil {
			return e
		}

		provider.logger.Sugar().Debugf("Store the new mapping for the key %s in Nuts, %v", key, string(val))

		return tx.Put(bucket, []byte(mappingKey), val, nutsdb.Persistent)
	})

	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Nuts, %v", err)
	}

	return err
}

// Set method will store the response in Nuts provider
func (provider *Nuts) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	err := provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucket, []byte(key), value, uint32(duration.Seconds()))
	})

	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Nuts, %v", err)
		return err
	}

	err = provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucket, []byte(StalePrefix+key), value, uint32((provider.stale + duration).Seconds()))
	})

	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Nuts, %v", err)
	}

	return nil
}

// Delete method will delete the response in Nuts provider if exists corresponding to key param
func (provider *Nuts) Delete(key string) {
	_ = provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.Delete(bucket, []byte(key))
	})
}

// DeleteMany method will delete the responses in Nuts provider if exists corresponding to the regex key param
func (provider *Nuts) DeleteMany(key string) {
	_ = provider.DB.Update(func(tx *nutsdb.Tx) error {
		if entries, err := tx.PrefixSearchScan(bucket, []byte(""), key, 0, nutsLimit); err != nil {
			return err
		} else {
			for _, entry := range entries {
				_ = tx.Delete(bucket, entry.Key)
			}
		}
		return nil
	})
}

// Init method will
func (provider *Nuts) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Nuts) Reset() error {
	return provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.DeleteBucket(1, bucket)
	})
}
