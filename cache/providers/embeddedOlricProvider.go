package providers

import (
	"context"
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
	dm     *olric.DMap
	db     *olric.Olric
	stale  time.Duration
	logger *zap.Logger
	ct     context.Context
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
		olricInstance.DMaps.MaxInuse = 512 << 20
	}

	started, cancel := context.WithCancel(context.Background())
	olricInstance.Started = func() {
		configuration.GetLogger().Sugar().Error("Embedded Olric is ready")
		defer cancel()
	}

	db, err := olric.New(olricInstance)
	if err != nil {
		return nil, err
	}

	ch := make(chan error, 1)
	defer func() {
		close(ch)
	}()

	go func(cdb *olric.Olric) {
		if err = cdb.Start(); err != nil {
			ch <- err
		}
	}(db)

	select {
	case err = <-ch:
	case <-started.Done():
	}
	dm, e := db.NewDMap("souin-map")

	configuration.GetLogger().Sugar().Info("Embedded Olric is ready for this node.")

	return &EmbeddedOlric{
		dm:     dm,
		db:     db,
		stale:  configuration.GetDefaultCache().GetStale(),
		logger: configuration.GetLogger(),
		ct:     context.Background(),
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
		provider.logger.Sugar().Errorf("An error occurred while trying to list keys in Olric: %s\n", err)
		return []string{}
	}

	keys := []string{}
	_ = c.Range(func(key string, _ interface{}) bool {
		keys = append(keys, key)
		return true
	})

	return keys
}

// Prefix method returns the populated response if exists, empty response then
func (provider *EmbeddedOlric) Prefix(key string, req *http.Request) []byte {
	c, err := provider.dm.Query(query.M{
		"$onKey": query.M{
			"$regexMatch": "^" + key + "({|$)",
		},
	})
	if c != nil {
		defer c.Close()
	}
	if err != nil {
		provider.logger.Sugar().Errorf("An error occurred while trying to retrieve data in Olric: %s\n", err)
		return []byte{}
	}

	res := []byte{}
	_ = c.Range(func(k string, v interface{}) bool {
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

	if err := provider.dm.PutEx(key, value, duration); err != nil {
		panic(err)
	}

	if err := provider.dm.PutEx(stalePrefix+key, value, provider.stale+duration); err != nil {
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
func (provider *EmbeddedOlric) Reset() error {
	return provider.db.Shutdown(provider.ct)
}

// Destruct method will reset or close provider
func (provider *EmbeddedOlric) Destruct() error {
	provider.logger.Sugar().Debug("Destruct current embedded olric...")
	return provider.Reset()
}

// GetDM method returns the embbeded instance dm property
func (provider *EmbeddedOlric) GetDM() *olric.DMap {
	return provider.dm
}
