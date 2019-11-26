package cache

import (
	"github.com/go-redis/redis"
	"time"
	"os"
	"strconv"
	"regexp"
)

func redisClientConnectionFactory() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_URL"),
		DB: 0,
		Password: "",
	})
}

func pathnameNotInRegex(pathname string) bool {
	b, _ := regexp.Match(os.Getenv("REGEX"), []byte(pathname))
	return !b
}

func getRequestInCache(pathname string, client *redis.Client) ReverseResponse {
	val2, err := client.Get(pathname).Result()

	if err != nil {
		return ReverseResponse{"", nil, nil}
	}

	return ReverseResponse{val2, nil, nil}
}

func deleteKey(key string, client *redis.Client) {
	client.Do("del", key)
}

func deleteKeys(regex string, client *redis.Client) {
	for _, i := range client.Keys(regex).Val() {
		client.Do("del", i)
	}
}

func setRequestInCache(pathname string, data []byte, client *redis.Client) {
	value, _ := strconv.Atoi(os.Getenv("TTL"))

	err := client.Set(pathname, string(data), time.Duration(value) * time.Second).Err()
	if err != nil {
		panic(err)
	}
}
