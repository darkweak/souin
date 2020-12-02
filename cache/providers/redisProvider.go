package providers

import (
	t "github.com/darkweak/souin/configurationtypes"
	redis "github.com/go-redis/redis/v8"
	"strconv"
	"time"
)

// Redis provider type
type Redis struct {
	*redis.Client
	t.AbstractConfigurationInterface
}

// RedisConnectionFactory function create new Redis instance
func RedisConnectionFactory(configuration t.AbstractConfigurationInterface) (*Redis, error) {
	return &Redis{
		redis.NewClient(&redis.Options{
			Addr:     configuration.GetDefaultCache().Redis.URL,
			DB:       0,
			Password: "",
		}),
		configuration,
	}, nil
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
	}
}

// Delete method will delete the response in Redis provider if exists corresponding to key param
func (provider *Redis) Delete(key string) {
	provider.Do(provider.Context(), "del", key)
}

// Init method will
func (provider *Redis) Init() error {
	return nil
}
