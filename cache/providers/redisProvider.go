package providers

import (
	"github.com/darkweak/souin/cache/keysaver"
	t "github.com/darkweak/souin/configurationtypes"
	redis "github.com/go-redis/redis/v8"
	"strconv"
	"time"
)

// Redis provider type
type Redis struct {
	*redis.Client
	keySaver *keysaver.ClearKey
}

// RedisConnectionFactory function create new Redis instance
func RedisConnectionFactory(configuration t.AbstractConfigurationInterface) (*Redis, error) {
	var keySaver *keysaver.ClearKey
	if configuration.GetAPI().Souin.Enable {
		keySaver = keysaver.NewClearKey()
		//TODO handle eviction on redis
	}

	return &Redis{
		redis.NewClient(&redis.Options{
			Addr:     configuration.GetDefaultCache().Redis.URL,
			DB:       0,
			Password: "",
		}),
		keySaver,
	}, nil
}

// ListKeys method returns the list of existing keys
func (provider *Redis) ListKeys() []string {
	if nil != provider.keySaver {
		return provider.keySaver.ListKeys()
	}
	return []string{}
}

// Get method returns the populated response if exists, empty response then
func (provider *Redis) Get(key string) []byte {
	val2, err := provider.Client.Get(provider.Context(), key).Result()

	if err != nil {
		return []byte{}
	}

	return []byte(val2)
}

// Set method will store the response in Redis provider
func (provider *Redis) Set(key string, value []byte, url t.URL, duration time.Duration) {
	if duration == 0 {
		ttl, _ := strconv.Atoi(url.TTL)
		duration = time.Duration(ttl)*time.Second
	}

	err := provider.Client.Set(provider.Context(), key, string(value), duration).Err()
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
func (provider *Redis) Delete(key string) {
	go func() {
		provider.Do(provider.Context(), "del", key)
		provider.keySaver.DelKey(key, 0)
	}()
}

// Init method will
func (provider *Redis) Init() error {
	return nil
}
