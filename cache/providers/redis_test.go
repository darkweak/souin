package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/configuration"
)

const REDISVALUE = "My first data"

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client := RedisConnectionFactory(*configuration.GetConfig())
	err := client.Set(client.Context(), "Test", string(REDISVALUE), time.Duration(10)*time.Second).Err()
	if err != nil {
		errors.GenerateError(t, "Impossible to set redis variable")
	}
	res, err := client.Get(client.Context(), "Test").Result()
	if err != nil {
		errors.GenerateError(t, "Retrieving data from redis")
	}
	if REDISVALUE != res {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, REDISVALUE))
	}
}
