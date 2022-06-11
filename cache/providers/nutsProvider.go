package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	t "github.com/darkweak/souin/configurationtypes"
	"github.com/imdario/mergo"
	"github.com/xujiajun/nutsdb"
)

// Nuts provider type
type Nuts struct {
	*nutsdb.DB
	stale time.Duration
}

const (
	bucket = "souin-bucket"
)

// NutsConnectionFactory function create new Nuts instance
func NutsConnectionFactory(c t.AbstractConfigurationInterface) (*Nuts, error) {
	dc := c.GetDefaultCache()
	nutsConfiguration := dc.GetNuts()
	nutsOptions := nutsdb.DefaultOptions
	nutsOptions.Dir = "/tmp/souin-nuts"
	if nutsConfiguration.Configuration != nil {
		var parsedNuts nutsdb.Options
		if b, e := json.Marshal(nutsConfiguration.Configuration); e == nil {
			if e = json.Unmarshal(b, &parsedNuts); e != nil {
				fmt.Println("Impossible to parse the configuration for the Nuts provider", e)
			}
		}

		if err := mergo.Merge(&nutsOptions, parsedNuts, mergo.WithOverride); err != nil {
			fmt.Println("An error occurred during the nutsOptions merge from the default options with your configuration.")
		}
	} else {
		nutsOptions.RWMode = nutsdb.MMap
		if nutsConfiguration.Path != "" {
			nutsOptions.Dir = nutsConfiguration.Path
		}
	}

	db, e := nutsdb.Open(nutsOptions)

	if e != nil {
		fmt.Println("Impossible to open the Nuts DB.", e)
	}

	i := &Nuts{DB: db, stale: dc.GetStale()}

	return i, nil
}

// ListKeys method returns the list of existing keys
func (provider *Nuts) ListKeys() []string {
	keys := []string{}

	e := provider.DB.View(func(tx *nutsdb.Tx) error {
		e, _ := tx.GetAll(bucket)
		for _, k := range e {
			keys = append(keys, string(k.Key))
		}
		return nil
	})

	if e != nil {
		return []string{}
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
func (provider *Nuts) Prefix(key string, req *http.Request) []byte {
	var result []byte

	_ = provider.DB.View(func(tx *nutsdb.Tx) error {
		prefix := []byte(key)

		if entries, _, err := tx.PrefixScan(bucket, prefix, 0, 50); err != nil {
			return err
		} else {
			for _, entry := range entries {
				if varyVoter(key, req, string(entry.Key)) {
					result = entry.Value
					return nil
				}
			}
		}
		return nil
	})

	return result
}

// Set method will store the response in Nuts provider
func (provider *Nuts) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	err := provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucket, []byte(key), value, uint32(duration.Seconds()))
	})

	if err != nil {
		panic(fmt.Sprintf("Impossible to set value into Nuts, %s", err))
	}

	err = provider.DB.Update(func(tx *nutsdb.Tx) error {
		return tx.Put(bucket, []byte(stalePrefix+key), value, uint32((provider.stale + duration).Seconds()))
	})

	if err != nil {
		panic(fmt.Sprintf("Impossible to set value into Nuts, %s", err))
	}
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
		if entries, _, err := tx.PrefixScan(bucket, []byte(key), 0, 100); err != nil {
			return err
		} else {
			for _, entry := range entries {
				tx.Delete(bucket, entry.Key)
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
