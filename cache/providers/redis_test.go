package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/darkweak/souin/errors"
)

const VALUE = "My first data"

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client := RedisConnectionFactory()
	err := client.Set("Test", string(VALUE), time.Duration(10)*time.Second).Err()
	if err != nil {
		errors.GenerateError(t, "Impossible to set redis variable")
	}
	res, err := client.Get("Test").Result()
	if err != nil {
		errors.GenerateError(t, "Retrieving data from redis")
	}
	if VALUE != res {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, VALUE))
	}
}
