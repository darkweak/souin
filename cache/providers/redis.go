package cache

import (
	"github.com/go-redis/redis"
	"time"
	"os"
	"strconv"
	"regexp"
	"github.com/darkweak/souin/cache"
)

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

// Check pathname is not in regex
func PathnameNotInRegex(pathname string) bool {
	b, _ := regexp.Match(os.Getenv("REGEX"), []byte(pathname))
	return !b
}

func (client *Redis) GetRequestInCache(pathname string) cache.ReverseResponse {
	val2, err := client.Get(pathname).Result()

	if err != nil {
		return cache.ReverseResponse{"", nil, nil}
	}

	return cache.ReverseResponse{val2, nil, nil}
}

func (client *Redis) DeleteKey(key string) {
	client.Do("del", key)
}

func (client *Redis) DeleteKeys(regex string) {
	for _, i := range client.Keys(regex).Val() {
		client.Do("del", i)
	}
}

func (client *Redis) SetRequestInCache(pathname string, data []byte) {
	value, _ := strconv.Atoi(os.Getenv("TTL"))

	err := client.Set(pathname, string(data), time.Duration(value)*time.Second).Err()
	if err != nil {
		panic(err)
	}
}
