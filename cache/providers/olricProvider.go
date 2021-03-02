package providers

import (
	"context"
	"fmt"
	"github.com/buraksezer/olric"
	"github.com/buraksezer/olric/config"
	"github.com/darkweak/souin/cache/keysaver"
	t "github.com/darkweak/souin/configurationtypes"
	"strconv"
	"time"
)

// Olric provider type
type Olric struct {
	*olric.Olric
	dm *olric.DMap
	keySaver *keysaver.ClearKey
}

// OlricConnectionFactory function create new Olric instance
func OlricConnectionFactory(configuration t.AbstractConfigurationInterface) (*Olric, error) {
	var keySaver *keysaver.ClearKey
	if configuration.GetAPI().Souin.Enable {
		keySaver = keysaver.NewClearKey()
	}
	db, err := olric.New(config.New("local"))
	if err != nil {
		panic(err)
	}

	return &Olric{
		db,
		nil,
		keySaver,
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Olric) ListKeys() []string {
	if nil != provider.keySaver {
		return provider.keySaver.ListKeys()
	}
	return []string{}
}

// Get method returns the populated response if exists, empty response then
func (provider *Olric) Get(key string) []byte {
	val2, err := provider.dm.Get(key)

	if err != nil {
		return []byte{}
	}

	return val2.([]byte)
}

// Set method will store the response in Redis provider
func (provider *Olric) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		ttl, _ := strconv.Atoi(url.TTL)
		duration = time.Duration(ttl)*time.Second
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
func (provider *Olric) Delete(key string) {
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

// Init method will initialize Olric provider if needed
func (provider *Olric) Init() error {
	c := make(chan interface{})

	go func() {
		defer func() {
			time.Sleep(2 * time.Second)
			c<-1
		}()
		go func() {
			_ = provider.Olric.Start()
		}()
	}()

	_ = <-c
	dm, _ := provider.Olric.NewDMap("souin-map")

	provider.dm = dm
	return nil
}

// Reset method will reset or close provider
func (provider *Olric) Reset() {
	e := provider.Olric.Shutdown(context.Background())
	fmt.Println(e)
}
