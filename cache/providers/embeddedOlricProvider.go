package providers

import (
	"context"
	"fmt"
	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/darkweak/souin/cache/keysaver"
	t "github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"time"
)

// EmbeddedOlric provider type
type EmbeddedOlric struct {
	dm       *olric.DMap
	db       *olric.Olric
	keySaver *keysaver.ClearKey
}

func tryToLoadConfiguration(olricInstance *config.Config, olricConfiguration t.CacheProvider, logger *zap.Logger) (*config.Config, bool) {
	var e error
	isAlreadyLoaded := false
	if olricConfiguration.Configuration == nil && olricConfiguration.Path != "" {
		if olricInstance, e = config.Load(olricConfiguration.Path); e == nil {
			isAlreadyLoaded = true
		}
	} else if olricConfiguration.Configuration != nil {
		tmpFile := "/tmp/souin-olric.yml"
		yamlConfig, e := yaml.Marshal(olricConfiguration.Configuration)
		defer func() {
			if e = os.RemoveAll(tmpFile); e != nil {
				logger.Error("Impossible to remove the temporary file")
			}
		}()
		if e = ioutil.WriteFile(
			tmpFile,
			yamlConfig,
			0644,
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
	var keySaver *keysaver.ClearKey
	if configuration.GetAPI().Souin.Enable {
		keySaver = keysaver.NewClearKey()
	}

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
			fmt.Println(fmt.Sprintf("Impossible to start the embedded Olric instance: %v", err))
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
		keySaver,
	}, e
}

// ListKeys method returns the list of existing keys
func (provider *EmbeddedOlric) ListKeys() []string {
	if nil != provider.keySaver {
		return provider.keySaver.ListKeys()
	}
	return []string{}
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
		ttl, err := time.ParseDuration(url.TTL)
		if err != nil {
			ttl = 0
			fmt.Println(err)
		}
		duration = ttl
	}

	err := provider.dm.PutEx(key, value, duration)
	if err != nil {
		panic(err)
	} else {
		go func() {
			if nil != provider.keySaver {
				provider.keySaver.AddKey(key)
			}
		}()
	}
}

// Delete method will delete the response in EmbeddedOlric provider if exists corresponding to key param
func (provider *EmbeddedOlric) Delete(key string) {
	go func() {
		err := provider.dm.Delete(key)
		if err != nil {
			panic(err)
		} else {
			go func() {
				if nil != provider.keySaver {
					provider.keySaver.DelKey(key, 0)
				}
			}()
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
