package cache

import (
	"testing"
	"time"
	"fmt"
	"github.com/darkweak/souin/cache"
)

const VALUE = "My first data"

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client := RedisConnectionFactory()
	err := client.Set("Test", string(VALUE), time.Duration(10)*time.Second).Err()
	if err != nil {
		cache.GenerateError(t, "Impossible to set redis variable")
	}
	res, err := client.Get("Test").Result()
	if err != nil {
		cache.GenerateError(t, "Retrieving data from redis")
	}
	if VALUE != res {
		cache.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, VALUE))
	}
}
