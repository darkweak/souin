package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
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

var enabledBadgerInstances = make(map[string]*Badger)

// BadgerConnectionFactory function create new Badger instance
func BadgerConnectionFactory(c t.AbstractConfigurationInterface) (*Badger, error) {
	dc := c.GetDefaultCache()
	badgerConfiguration := dc.GetBadger()
	badgerOptions := badger.DefaultOptions(badgerConfiguration.Path)
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
	} else if badgerConfiguration.Path == "" {
		badgerOptions = badgerOptions.WithInMemory(true)
	}

	uid := badgerOptions.Dir + badgerOptions.ValueDir
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

// ListKeys method returns the list of existing keys
func (provider *Badger) ListKeys() []string {
	keys := []string{}

	e := provider.DB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			keys = append(keys, string(it.Item().Key()))
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

	_ = item.Value(func(val []byte) error {
		result = val
		return nil
	})

	return result
}

// Prefix method returns the populated response if exists, empty response then
func (provider *Badger) Prefix(key string, req *http.Request) []byte {
	var result []byte

	_ = provider.DB.View(func(txn *badger.Txn) error {
		prefix := []byte(key)
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if varyVoter(key, req, string(it.Item().Key())) {
				_ = it.Item().Value(func(val []byte) error {
					result = val
					return nil
				})
			}
		}
		return nil
	})

	return result
}

// Set method will store the response in Badger provider
func (provider *Badger) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	err := provider.DB.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry([]byte(key), value).WithTTL(duration))
	})

	if err != nil {
		panic(fmt.Sprintf("Impossible to set value into Badger, %s", err))
	}

	err = provider.DB.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(badger.NewEntry([]byte(stalePrefix+key), value).WithTTL(provider.stale + duration))
	})

	if err != nil {
		panic(fmt.Sprintf("Impossible to set value into Badger, %s", err))
	}
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
