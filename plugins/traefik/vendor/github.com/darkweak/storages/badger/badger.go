//go:build !wasm && !wasi

package badger

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/darkweak/storages/core"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

// sanitizeProperties coerces string scalars to the numeric and boolean
// badger.Options fields they address (Caddyfile dispensers tokenize every
// scalar as a string, so those options would otherwise fail to unmarshal and
// be dropped). Values that don't parse are left unchanged so the subsequent
// unmarshal reports them as before.
func sanitizeProperties(configMap map[string]interface{}) map[string]interface{} {
	boolFields := []string{
		"SyncWrites", "ReadOnly", "InMemory", "MetricsEnabled", "CompactL0OnClose",
		"LmaxCompaction", "VerifyValueChecksum", "BypassLockGuard", "DetectConflicts",
	}
	for _, field := range boolFields {
		if s, ok := configMap[field].(string); ok {
			if v, err := strconv.ParseBool(s); err == nil {
				configMap[field] = v
			}
		}
	}

	intFields := []string{
		"NumVersionsToKeep", "NumGoroutines", "LevelSizeMultiplier", "TableSizeMultiplier",
		"MaxLevels", "MemTableSize", "BaseTableSize", "BaseLevelSize", "ValueThreshold",
		"NumMemtables", "BlockSize", "BlockCacheSize", "IndexCacheSize", "NumLevelZeroTables",
		"NumLevelZeroTablesStall", "ValueLogFileSize", "NumCompactors", "ZSTDCompressionLevel",
		"EncryptionKeyRotationDuration", "NamespaceOffset", "ChecksumVerificationMode",
	}
	for _, field := range intFields {
		if s, ok := configMap[field].(string); ok {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				configMap[field] = v
			}
		}
	}

	uintFields := []string{"ValueLogMaxEntries", "Compression", "ExternalMagicVersion"}
	for _, field := range uintFields {
		if s, ok := configMap[field].(string); ok {
			if v, err := strconv.ParseUint(s, 10, 64); err == nil {
				configMap[field] = v
			}
		}
	}

	floatFields := []string{"VLogPercentile", "BloomFalsePositive"}
	for _, field := range floatFields {
		if s, ok := configMap[field].(string); ok {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				configMap[field] = v
			}
		}
	}

	return configMap
}

// Badger provider type.
type Badger struct {
	*badger.DB

	stale  time.Duration
	logger core.Logger
}

var (
	enabledBadgerInstances               = sync.Map{}
	_                      badger.Logger = (*badgerLogger)(nil)
)

type badgerLogger struct {
	*zap.SugaredLogger
}

func (b *badgerLogger) Warningf(msg string, params ...interface{}) {
	b.Warnf(msg, params...)
}

// Factory function create new Badger instance.
func Factory(badgerConfiguration core.CacheProvider, logger core.Logger, stale time.Duration) (core.Storer, error) {
	badgerOptions := badger.DefaultOptions(badgerConfiguration.Path)
	badgerOptions.SyncWrites = true
	badgerOptions.MemTableSize = 64 << 22

	if badgerConfiguration.Configuration != nil {
		var parsedBadger badger.Options

		if configMap, ok := badgerConfiguration.Configuration.(map[string]interface{}); ok {
			badgerConfiguration.Configuration = sanitizeProperties(configMap)
		}

		if b, e := json.Marshal(badgerConfiguration.Configuration); e == nil {
			if e = json.Unmarshal(b, &parsedBadger); e != nil {
				logger.Error("Impossible to parse the configuration for the default provider (Badger)", e)
			}
		}

		if err := mergo.Merge(&badgerOptions, parsedBadger, mergo.WithOverride); err != nil {
			logger.Error("An error occurred during the badgerOptions merge from the default options with your configuration.")
		}

		if !badgerOptions.InMemory {
			if badgerOptions.Dir == "" {
				badgerOptions.Dir = "souin_dir"
			}

			if badgerOptions.ValueDir == "" {
				badgerOptions.ValueDir = badgerOptions.Dir
			}
		}
	} else if badgerConfiguration.Path == "" {
		badgerOptions = badgerOptions.WithInMemory(true)
	}

	zapLogger, ok := logger.(*zap.SugaredLogger)
	if ok {
		badgerOptions.Logger = &badgerLogger{SugaredLogger: zapLogger}
	}

	uid := badgerOptions.Dir + badgerOptions.ValueDir + stale.String()

	if instance, ok := enabledBadgerInstances.Load(uid); ok {
		return instance.(*Badger), nil
	}

	db, e := badger.Open(badgerOptions)
	if e != nil {
		logger.Error("Impossible to open the Badger DB.", e)
	}

	i := &Badger{DB: db, logger: logger, stale: stale}
	enabledBadgerInstances.Store(uid, i)

	return i, nil
}

// Name returns the storer name.
func (provider *Badger) Name() string {
	return "BADGER"
}

// Uuid returns an unique identifier.
func (provider *Badger) Uuid() string {
	return fmt.Sprintf(
		"%s-%s-%s",
		provider.DB.Opts().Dir,
		provider.DB.Opts().ValueDir,
		provider.stale,
	)
}

// MapKeys method returns a map with the key and value.
func (provider *Badger) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}

	_ = provider.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		iterator := txn.NewIterator(opts)
		p := []byte(prefix)

		defer iterator.Close()

		for iterator.Seek(p); iterator.ValidForPrefix(p); iterator.Next() {
			val, _ := iterator.Item().ValueCopy(nil)
			k, _ := strings.CutPrefix(string(iterator.Item().Key()), prefix)
			keys[k] = string(val)
		}

		return nil
	})

	return keys
}

// ListKeys method returns the list of existing keys.
func (provider *Badger) ListKeys() []string {
	keys := []string{}

	err := provider.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)

		defer it.Close()

		for it.Seek([]byte(core.MappingKeyPrefix)); it.ValidForPrefix([]byte(core.MappingKeyPrefix)); it.Next() {
			val, _ := it.Item().ValueCopy(nil)

			mapping, err := core.DecodeMapping(val)
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

// Get method returns the populated response if exists, empty response then.
func (provider *Badger) Get(key string) []byte {
	var result []byte

	_ = provider.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		// The value is only valid within the transaction, copy it there.
		result, _ = item.ValueCopy(nil)

		return nil
	})

	return result
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Badger) GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response) {
	_ = provider.View(func(tx *badger.Txn) error {
		result, err := tx.Get([]byte(core.MappingKeyPrefix + key))
		if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}

		var val []byte

		if result != nil {
			val, _ = result.ValueCopy(nil)
		}

		fresh, stale, err = core.MappingElection(provider, val, req, validator, provider.logger)

		return err
	})

	return
}

// SetMultiLevel tries to store the key with the given value and update the mapping key to store metadata.
func (provider *Badger) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error {
	now := time.Now()

	compressed, err := core.CompressValue(value)
	if err != nil {
		provider.logger.Errorf("Impossible to compress the key %s into Badger, %v", variedKey, err)

		return err
	}

	err = provider.Update(func(btx *badger.Txn) error {
		var err error

		err = btx.SetEntry(badger.NewEntry([]byte(variedKey), compressed).WithTTL(duration + provider.stale))
		if err != nil {
			provider.logger.Errorf("Impossible to set the key %s into Badger, %v", variedKey, err)

			return err
		}

		mappingKey := core.MappingKeyPrefix + baseKey
		item, err := btx.Get([]byte(mappingKey))

		if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
			provider.logger.Errorf("Impossible to get the base key %s in Badger, %v", mappingKey, err)

			return err
		}

		var val []byte

		if item != nil {
			val, _ = item.ValueCopy(nil)
		}

		val, err = core.MappingUpdater(variedKey, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag, realKey)
		if err != nil {
			provider.logger.Errorf("Impossible to update the mapping for the key %s in Badger, %v", variedKey, err)

			return err
		}

		provider.logger.Debugf("Store the new mapping for the key %s in Badger", variedKey)

		return btx.SetEntry(badger.NewEntry([]byte(mappingKey), val))
	})
	if err != nil {
		provider.logger.Errorf("Impossible to set value into Badger, %v", err)
	}

	return err
}

// Set method will store the response in Badger provider.
func (provider *Badger) Set(key string, value []byte, duration time.Duration) error {
	err := provider.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry([]byte(key), value).WithTTL(duration))
	})
	if err != nil {
		provider.logger.Errorf("Impossible to set value into Badger, %v", err)
	}

	return err
}

// Delete method will delete the response in Badger provider if exists corresponding to key param.
func (provider *Badger) Delete(key string) {
	_ = provider.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

// DeleteMany method will delete the responses in Badger provider if exists corresponding to the regex key param.
func (provider *Badger) DeleteMany(key string) {
	rgKey, e := regexp.Compile(key)
	if e != nil {
		return
	}

	_ = provider.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)

		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			k := string(it.Item().Key())
			if rgKey.MatchString(k) {
				provider.Delete(k)
			}
		}

		return nil
	})
}

// Init method will.
func (provider *Badger) Init() error {
	return nil
}

// Reset method will reset or close provider.
func (provider *Badger) Reset() error {
	var err error
	// Close the DB connection
	if provider.DB != nil {
		err = provider.Close()
	}
	// Only delete this instance from the cache
	enabledBadgerInstances.Delete(provider.Uuid())

	return err
}
