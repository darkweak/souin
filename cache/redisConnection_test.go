package cache

import (
	"testing"
	"time"
	"fmt"
)

const VALUE = "My first data"

func populateRedisWithFakeData() {
	client := redisClientConnectionFactory()
	duration := time.Duration(120) * time.Second
	basePath := "/testing"
	domain := "domain.com"

	client.Set(domain+basePath, "testing value is here for "+basePath, duration)
	for i := 0; i < 25; i++ {
		client.Set(domain+basePath+"/"+string(i), "testing value is here for my first init of "+basePath+"/"+string(i), duration)
	}
}

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client := redisClientConnectionFactory()
	err := client.Set("Test", string(VALUE), time.Duration(10)*time.Second).Err()
	if err != nil {
		generateError(t, "Impossible to set redis variable")
	}
	res, err := client.Get("Test").Result()
	if err != nil {
		generateError(t, "Retrieving data from redis")
	}
	if VALUE != res {
		generateError(t, fmt.Sprintf("%s not corresponding to %s", res, VALUE))
	}
}
