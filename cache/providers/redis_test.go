package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/configuration"
)

const VALUE = "My first data"

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client := RedisConnectionFactory(*configuration.GetConfig())
	err := client.Set(client.Context(), "Test", string(VALUE), time.Duration(10)*time.Second).Err()
	if err != nil {
		errors.GenerateError(t, "Impossible to set redis variable")
	}
	res, err := client.Get(client.Context(), "Test").Result()
	if err != nil {
		errors.GenerateError(t, "Retrieving data from redis")
	}
	if VALUE != res {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, VALUE))
	}
}
