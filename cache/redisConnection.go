package cache

import (
	"github.com/go-redis/redis"
	"time"
	"os"
	"strconv"
	"regexp"
)

type Redis struct {
	*redis.Client
}

func redisConnectionFactory() *Redis {
	return &Redis{
		redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_URL"),
			DB:       0,
			Password: "",
		}),
	}
}

func pathnameNotInRegex(pathname string) bool {
	b, _ := regexp.Match(os.Getenv("REGEX"), []byte(pathname))
	return !b
}

func (client *Redis) getRequestInCache(pathname string) ReverseResponse {
	val2, err := client.Get(pathname).Result()

	if err != nil {
		return ReverseResponse{"", nil, nil}
	}

	return ReverseResponse{val2, nil, nil}
}

func (client *Redis) deleteKey(key string) {
	Redis.Do("del", key)
}

func (client *Redis) deleteKeys(regex string) {
	for _, i := range client.Keys(regex).Val() {
		client.Do("del", i)
	}
}

func (client *Redis) setRequestInCache(pathname string, data []byte) {
	value, _ := strconv.Atoi(os.Getenv("TTL"))

	err := client.Set(pathname, string(data), time.Duration(value)*time.Second).Err()
	if err != nil {
		panic(err)
	}
}
