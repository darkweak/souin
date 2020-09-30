package providers

import (
	"strconv"
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	redis "github.com/go-redis/redis/v8"
)

// Redis provider type
type Redis struct {
	*redis.Client
	configuration.Configuration
}

// RedisConnectionFactory function create new Redis instance
func RedisConnectionFactory(configuration configuration.Configuration) *Redis {
	return &Redis{
		redis.NewClient(&redis.Options{
			Addr:     configuration.DefaultCache.Redis.URL,
			DB:       0,
			Password: "",
		}),
		configuration,
	}
}

// GetRequestInCache method returns the populated response if exists, empty response then
func (provider *Redis) GetRequestInCache(key string) types.ReverseResponse {
	val2, err := provider.Get(provider.Context(), key).Result()

	if err != nil {
		return types.ReverseResponse{Response: "", Proxy: nil, Request: nil}
	}

	return types.ReverseResponse{Response: val2, Proxy: nil, Request: nil}
}

// SetRequestInCache method will store the response in Redis provider
func (provider *Redis) SetRequestInCache(key string, value []byte, url configuration.URL) {
	ttl, _ := strconv.Atoi(url.TTL)

	err := provider.Set(provider.Context(), key, string(value), time.Duration(ttl)*time.Second).Err()
	if err != nil {
		panic(err)
	}
}

// DeleteRequestInCache method will delete the response in Redis provider if exists corresponding to key param
func (provider *Redis) DeleteRequestInCache(key string) {
	provider.Do(provider.Context(), "del", key)
}

// Init method will
func (provider *Redis) Init() error {
	return nil
}
