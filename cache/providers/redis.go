package providers

import (
	"strconv"
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/go-redis/redis"
	"github.com/darkweak/souin/configuration"
)

// Redis provider type
type Redis struct {
	*redis.Client;
	configuration.Configuration;
}

// RedisConnectionFactory function create new Redis instance
func RedisConnectionFactory(configuration configuration.Configuration) *Redis {
	return &Redis{
		redis.NewClient(&redis.Options{
			Addr:     configuration.Redis.Url,
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
func (provider *Redis) SetRequestInCache(key string, value []byte) {
	ttl, _ := strconv.Atoi(provider.Configuration.TTL)

	err := provider.Set(provider.Context(), key, string(value), time.Duration(ttl)*time.Second).Err()
	if err != nil {
		panic(err)
	}
}

// DeleteRequestInCache method will delete the response in Redis provider if exists corresponding to key param
func (provider *Redis) DeleteRequestInCache(key string) {
	provider.Do(provider.Context(), "del", key)
}

// DeleteManyRequestInCache method will delete the response in Redis provider if exists corresponding to regex param
func (provider *Redis) DeleteManyRequestInCache(regex string) {
	for _, i := range provider.Keys(provider.Context(), regex).Val() {
		provider.Do(provider.Context(), provider, "del", i)
	}
}

// Init method will
func (provider *Redis) Init() {}
