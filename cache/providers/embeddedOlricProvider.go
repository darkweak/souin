package providers

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/buraksezer/olric/query"
	t "github.com/darkweak/souin/configurationtypes"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// EmbeddedOlric provider type
type EmbeddedOlric struct {
	dm *olric.DMap
	db *olric.Olric
}

func tryToLoadConfiguration(olricInstance *config.Config, olricConfiguration t.CacheProvider, logger *zap.Logger) (*config.Config, bool) {
	var e error
	isAlreadyLoaded := false
	if olricConfiguration.Configuration == nil && olricConfiguration.Path != "" {
		if olricInstance, e = config.Load(olricConfiguration.Path); e == nil {
			isAlreadyLoaded = true
		}
	} else if olricConfiguration.Configuration != nil {
		tmpFile := "/tmp/" + uuid.NewString() + ".yml"
		yamlConfig, e := yaml.Marshal(olricConfiguration.Configuration)
		defer func() {
			if e = os.RemoveAll(tmpFile); e != nil {
				logger.Error("Impossible to remove the temporary file")
			}
		}()
		if e = ioutil.WriteFile(
			tmpFile,
			yamlConfig,
			0600,
		); e != nil {
			logger.Error("Impossible to create the embedded Olric config from the given one")
		}

		if olricInstance, e = config.Load(tmpFile); e == nil {
			isAlreadyLoaded = true
		} else {
			logger.Error("Impossible to create the embedded Olric config from the given one")
		}
	}

	return olricInstance, isAlreadyLoaded
}

// EmbeddedOlricConnectionFactory function create new EmbeddedOlric instance
func EmbeddedOlricConnectionFactory(configuration t.AbstractConfigurationInterface) (*EmbeddedOlric, error) {
	var olricInstance *config.Config
	loaded := false

	if olricInstance, loaded = tryToLoadConfiguration(olricInstance, configuration.GetDefaultCache().GetOlric(), configuration.GetLogger()); !loaded {
		olricInstance = config.New("local")
		olricInstance.Cache.MaxInuse = 512 << 20
	}

	started, cancel := context.WithCancel(context.Background())
	olricInstance.Started = func() {
		defer cancel()
		configuration.GetLogger().Info("Embedded Olric is ready")
	}

	db, err := olric.New(olricInstance)
	if err != nil {
		return nil, err
	}

	ch := make(chan error, 1)
	go func() {
		if err = db.Start(); err != nil {
			fmt.Printf("Impossible to start the embedded Olric instance: %v\n", err)
			ch <- err
		}
	}()

	select {
	case err = <-ch:
		return nil, err
	case <-started.Done():
	}
	dm, e := db.NewDMap("souin-map")

	return &EmbeddedOlric{
		dm,
		db,
	}, e
}

// ListKeys method returns the list of existing keys
func (provider *EmbeddedOlric) ListKeys() []string {
	c, err := provider.dm.Query(query.M{
		"$onKey": query.M{
			"$regexMatch": "",
			"$options": query.M{
				"$onValue": query.M{
					"$ignore": true,
				},
			},
		},
	})
	if c != nil {
		defer c.Close()
	}
	if err != nil {
		fmt.Printf("An error occurred while trying to list keys in Olric: %s\n", err)
		return []string{}
	}

	keys := []string{}
	err = c.Range(func(key string, _ interface{}) bool {
		keys = append(keys, key)
		return true
	})

	return keys
}

// Prefix method returns the populated response if exists, empty response then
func (provider *EmbeddedOlric) Prefix(key string, req *http.Request) []byte {
	c, err := provider.dm.Query(query.M{
		"$onKey": query.M{
			"$regexMatch": "^" + key,
		},
	})
	if c != nil {
		defer c.Close()
	}
	if err != nil {
		fmt.Printf("An error occurred while trying to retrieve data in Olric: %s\n", err)
		return []byte{}
	}

	res := []byte{}
	err = c.Range(func(k string, v interface{}) bool {
		if varyVoter(key, req, k) {
			res = v.([]byte)
			return false
		}

		return true
	})

	return res
}

// Get method returns the populated response if exists, empty response then
func (provider *EmbeddedOlric) Get(key string) []byte {
	val2, err := provider.dm.Get(key)

	if err != nil {
		return []byte{}
	}

	return val2.([]byte)
}

// Set method will store the response in EmbeddedOlric provider
func (provider *EmbeddedOlric) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		duration = url.TTL.Duration
	}

	err := provider.dm.PutEx(key, value, duration)
	if err != nil {
		panic(err)
	}
}

// Delete method will delete the response in EmbeddedOlric provider if exists corresponding to key param
func (provider *EmbeddedOlric) Delete(key string) {
	go func() {
		err := provider.dm.Delete(key)
		if err != nil {
			panic(err)
		}
	}()
}

// DeleteMany method will delete the responses in EmbeddedOlric provider if exists corresponding to the regex key param
func (provider *EmbeddedOlric) DeleteMany(key string) {
	go func() {
		c, err := provider.dm.Query(query.M{
			"$onKey": query.M{
				"$regexMatch": key,
				"$options": query.M{
					"$onValue": query.M{
						"$ignore": true,
					},
				},
			},
		})

		if c == nil || err != nil {
			return
		}

		err = c.Range(func(key string, _ interface{}) bool {
			provider.Delete(key)
			return true
		})

		if err != nil {
			panic(err)
		}
	}()
}

// Init method will initialize EmbeddedOlric provider if needed
func (provider *EmbeddedOlric) Init() error {
	return nil
}

// Reset method will reset or close provider
func (provider *EmbeddedOlric) Reset() {
	_ = provider.db.Shutdown(context.Background())
}
