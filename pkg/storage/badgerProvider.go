package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/imdario/mergo"
	"go.uber.org/zap"
)

// Badger provider type
type Badger struct {
	*badger.DB
	stale  time.Duration
	logger *zap.Logger
}

var (
	enabledBadgerInstances               = make(map[string]*Badger)
	_                      badger.Logger = (*badgerLogger)(nil)
)

type badgerLogger struct {
	*zap.SugaredLogger
}

func (b *badgerLogger) Warningf(msg string, params ...interface{}) {
	b.SugaredLogger.Warnf(msg, params...)
}

// BadgerConnectionFactory function create new Badger instance
func BadgerConnectionFactory(c t.AbstractConfigurationInterface) (types.Storer, error) {
	dc := c.GetDefaultCache()
	badgerConfiguration := dc.GetBadger()
	badgerOptions := badger.DefaultOptions(badgerConfiguration.Path)
	badgerOptions.SyncWrites = true
	badgerOptions.MemTableSize = 64 << 22
	if badgerConfiguration.Configuration != nil {
		var parsedBadger badger.Options
		if b, e := json.Marshal(badgerConfiguration.Configuration); e == nil {
			if e = json.Unmarshal(b, &parsedBadger); e != nil {
				c.GetLogger().Sugar().Error("Impossible to parse the configuration for the default provider (Badger)", e)
			}
		}

		if err := mergo.Merge(&badgerOptions, parsedBadger, mergo.WithOverride); err != nil {
			c.GetLogger().Sugar().Error("An error occurred during the badgerOptions merge from the default options with your configuration.")
		}
		if badgerOptions.InMemory {
			badgerOptions.Dir = ""
			badgerOptions.ValueDir = ""
		} else {
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

	badgerOptions.Logger = &badgerLogger{SugaredLogger: c.GetLogger().Sugar()}
	uid := badgerOptions.Dir + badgerOptions.ValueDir + dc.GetStale().String()
	if i, ok := enabledBadgerInstances[uid]; ok {
		return i, nil
	}

	db, e := badger.Open(badgerOptions)

	if e != nil {
		c.GetLogger().Sugar().Error("Impossible to open the Badger DB.", e)
	}

	i := &Badger{DB: db, logger: c.GetLogger(), stale: dc.GetStale()}
	enabledBadgerInstances[uid] = i

	return i, nil
}

// Name returns the storer name
func (provider *Badger) Name() string {
	return "BADGER"
}

// MapKeys method returns a map with the key and value
func (provider *Badger) MapKeys(prefix string) map[string]string {
	keys := map[string]string{}

	_ = provider.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		p := []byte(prefix)
		for it.Seek(p); it.ValidForPrefix(p); it.Next() {
			_ = it.Item().Value(func(val []byte) error {
				k, _ := strings.CutPrefix(string(it.Item().Key()), prefix)
				keys[k] = string(val)

				return nil
			})
		}
		return nil
	})

	return keys
}

// ListKeys method returns the list of existing keys
func (provider *Badger) ListKeys() []string {
	keys := []string{}

	e := provider.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			key := string(it.Item().Key())
			if !strings.Contains(key, surrogatePrefix) {
				keys = append(keys, key)
			}
		}
		return nil
	})

	if e != nil {
		return []string{}
	}

	return keys
}

// Get method returns the populated response if exists, empty response then
func (provider *Badger) Get(key string) []byte {
	var item *badger.Item
	var result []byte

	e := provider.DB.View(func(txn *badger.Txn) error {
		i, err := txn.Get([]byte(key))
		item = i
		return err
	})

	if e == badger.ErrKeyNotFound {
		return result
	}

	if item != nil {
		_ = item.Value(func(val []byte) error {
			result = val
			return nil
		})
	}

	return result
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Badger) Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response {
	var result *http.Response

	_ = provider.DB.View(func(txn *badger.Txn) error {
		prefix := []byte(key)
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if varyVoter(key, req, string(it.Item().Key())) {
				_ = it.Item().Value(func(val []byte) error {
					if res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(val)), req); err == nil {
						rfc.ValidateETag(res, validator)
						if validator.Matched {
							provider.logger.Sugar().Debugf("The stored key %s matched the current iteration key ETag %+v", it.Item().Key(), validator)
							result = res
						} else {
							provider.logger.Sugar().Debugf("The stored key %s didn't match the current iteration key ETag %+v", it.Item().Key(), validator)
						}
					} else {
						provider.logger.Sugar().Errorf("An error occured while reading response for the key %s: %v", it.Item().Key(), err)
					}

					return nil
				})
			}
		}
		return nil
	})

	return result
}

// GetMultiLevel tries to load the key and check if one of linked keys is a fresh/stale candidate.
func (provider *Badger) GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response) {
	_ = provider.DB.View(func(tx *badger.Txn) error {
		i, e := tx.Get([]byte(mappingKeyPrefix + key))
		if e != nil && !errors.Is(e, badger.ErrKeyNotFound) {
			return e
		}

		var val []byte
		if i != nil {
			_ = i.Value(func(b []byte) error {
				val = b

				return nil
			})
		}
		fresh, stale, e = mappingElection(provider, val, req, validator, provider.logger)

		return e
	})

	return
}

// SetMultiLevel tries to store the keywith the given value and update the mapping key to store metadata.
func (provider *Badger) SetMultiLevel(baseKey, key string, value []byte, variedHeaders http.Header, etag string, duration time.Duration) error {
	now := time.Now()

	err := provider.DB.Update(func(tx *badger.Txn) error {
		var e error
		e = tx.SetEntry(badger.NewEntry([]byte(key), value).WithTTL(duration + provider.stale))
		if e != nil {
			provider.logger.Sugar().Errorf("Impossible to set the key %s into Badger, %v", key, e)
			return e
		}

		mappingKey := mappingKeyPrefix + baseKey
		item, e := tx.Get([]byte(mappingKey))
		if e != nil && !errors.Is(e, badger.ErrKeyNotFound) {
			provider.logger.Sugar().Errorf("Impossible to get the base key %s in Badger, %v", mappingKey, e)
			return e
		}

		var val []byte
		if item != nil {
			_ = item.Value(func(b []byte) error {
				val = b

				return nil
			})
		}

		val, e = mappingUpdater(key, val, provider.logger, now, now.Add(duration), now.Add(duration+provider.stale), variedHeaders, etag)
		if e != nil {
			return e
		}

		provider.logger.Sugar().Errorf("Store the new mapping for the key %s in Badger, %v", key, string(val))
		return tx.SetEntry(badger.NewEntry([]byte(mappingKey), val))
	})

	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Badger, %v", err)
	}

	return err
}

// Set method will store the response in Badger provider
func (provider *Badger) Set(key string, value []byte, url t.URL, duration time.Duration) error {
	err := provider.DB.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry([]byte(key), value).WithTTL(duration))
	})

	if err != nil {
		provider.logger.Sugar().Errorf("Impossible to set value into Badger, %v", err)
	}

	return err
}

// Delete method will delete the response in Badger provider if exists corresponding to key param
func (provider *Badger) Delete(key string) {
	_ = provider.DB.DropPrefix([]byte(key))
}

// DeleteMany method will delete the responses in Badger provider if exists corresponding to the regex key param
func (provider *Badger) DeleteMany(key string) {
	re, e := regexp.Compile(key)

	if e != nil {
		return
	}

	_ = provider.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			k := string(it.Item().Key())
			if re.MatchString(k) {
				provider.Delete(k)
			}
		}
		return nil
	})
}

// Init method will
func (provider *Badger) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *Badger) Reset() error {
	return provider.DB.DropAll()
}
