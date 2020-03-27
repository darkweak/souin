package providers

import (
	"strconv"
	"os"
	"time"
	"github.com/go-redis/redis"
	"github.com/darkweak/souin/cache/types"
)

// Redis provider type
type Redis struct {
	*redis.Client
}

// RedisConnectionFactory function create new Redis instance
func RedisConnectionFactory() *Redis {
	return &Redis{
		redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_URL"),
			DB:       0,
			Password: "",
		}),
	}
}

// GetRequestInCache method returns the populated response if exists, empty response then
func (provider *Redis) GetRequestInCache(key string) types.ReverseResponse {
	val2, err := provider.Get(key).Result()

	if err != nil {
		return types.ReverseResponse{Response: "", Proxy: nil, Request: nil}
	}

	return types.ReverseResponse{Response: val2, Proxy: nil, Request: nil}
}

// SetRequestInCache method will store the response in Redis provider
func (provider *Redis) SetRequestInCache(key string, value []byte) {
	ttl, _ := strconv.Atoi(os.Getenv("TTL"))

	err := provider.Set(key, string(value), time.Duration(ttl)*time.Second).Err()
	if err != nil {
		panic(err)
	}
}

// DeleteRequestInCache method will delete the response in Redis provider if exists corresponding to key param
func (provider *Redis) DeleteRequestInCache(key string) {
	provider.Do("del", key)
}

// DeleteManyRequestInCache method will delete the response in Redis provider if exists corresponding to regex param
func (provider *Redis) DeleteManyRequestInCache(regex string) {
	for _, i := range provider.Keys(regex).Val() {
		provider.Do("del", i)
	}
}

// Init method will
func (provider *Redis) Init() {}
