package providers

import (
	"context"
	"fmt"
	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/darkweak/souin/cache/keysaver"
	t "github.com/darkweak/souin/configurationtypes"
	"time"
)

// EmbeddedOlric provider type
type EmbeddedOlric struct {
	dm       *olric.DMap
	db       *olric.Olric
	keySaver *keysaver.ClearKey
}

// EmbeddedOlricConnectionFactory function create new EmbeddedOlric instance
func EmbeddedOlricConnectionFactory(configuration t.AbstractConfigurationInterface) (*EmbeddedOlric, error) {
	olricConfiguration := configuration.GetDefaultCache().GetOlric()
	var keySaver *keysaver.ClearKey
	if configuration.GetAPI().Souin.Enable {
		keySaver = keysaver.NewClearKey()
	}

	var olricInstance *config.Config
	var e error
	isAlreadyLoaded := false
	if olricInstance, e = config.Load(olricConfiguration.Path); e != nil {
		isAlreadyLoaded = true
	}

	if !isAlreadyLoaded {
		olricInstance = config.New("local")
		olricInstance.Cache.MaxInuse = 512 << 20
	}

	started, cancel := context.WithCancel(context.Background())
	olricInstance.Started = func() {
		defer cancel()
		fmt.Println("Embedded Olric is ready")
	}

	db, err := olric.New(olricInstance)
	if err != nil {
		return nil, err
	}

	go func() {
		if err = db.Start(); err != nil {
			fmt.Println(fmt.Sprintf("Impossible to start the embedded Olric instance: %v", err))
		}
	}()

	select {
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

// Set method will store the response in Redis provider
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

// Delete method will delete the response in Redis provider if exists corresponding to key param
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
