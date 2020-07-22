package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/configuration"
)

const REDISVALUE = "My first data"
const DELETABLEKEY = "MyDeletableKey"

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())
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

func TestRedis_GetRequestInCache(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())
	res := client.GetRequestInCache(NONEXISTENTKEY)
	if res.Response != "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestRedis_SetRequestInCache_OneByte(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())
	client.SetRequestInCache(BYTEKEY, []byte{65})
}

func TestRedis_GetRequestInCache_OneByte(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())
	res := client.GetRequestInCache(BYTEKEY)
	if res.Response == "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BYTEKEY))
	}
	a := string([]byte{65})

	if res.Response != a {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %v", res.Response, 65))
	}
}

func TestRedis_SetRequestInCache_MultipleKeys(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())

	for i:= 0; i < 10; i++ {
		err := client.Set(client.Context(), fmt.Sprintf("%s%v", DELETABLEKEY, i), string([]byte{65}), time.Duration(30)*time.Second).Err()
		if err != nil {
			errors.GenerateError(t, "Impossible to set redis variable")
		}
	}
}

func TestRedis_SetRequestInCache_ExistingKey(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())

	for i:= 0; i < 10; i++ {
		err := client.Set(client.Context(), BYTEKEY, "New value", time.Duration(10)*time.Second).Err()
		if err != nil {
			errors.GenerateError(t, "Impossible to set redis variable")
		}
	}
}

func TestRedis_DeleteRequestInCache(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())
	client.DeleteRequestInCache(BYTEKEY)
	if "" != client.GetRequestInCache(BYTEKEY).Response {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestRedis_Init(t *testing.T) {
	client := RedisConnectionFactory(configuration.GetConfig())
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to instantiate Redis provider")
	}
}
