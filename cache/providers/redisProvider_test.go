package providers

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/tests"

	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

const REDISVALUE = "My first data"

func getRedisClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return tests.GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.RedisConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error) {
			provider, _ := RedisConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

func TestRedisConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration(tests.RedisConfiguration)
	r, err := RedisConnectionFactory(c)

	if nil != err {
		errors.GenerateError(t, "Shouldn't have panic")
	}

	if nil == r {
		errors.GenerateError(t, "Redis should be instanciated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client, matchedURL := getRedisClientAndMatchedURL("Test")

	client.Set("Test", []byte(REDISVALUE), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get("Test")
	if res == nil || len(res) <= 0 {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", REDISVALUE))
	}
	if REDISVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", string(res), REDISVALUE))
	}
}

func TestRedis_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration(tests.RedisConfiguration)
	client, _ := RedisConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestRedis_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getRedisClientAndMatchedURL(BYTEKEY)
	client.Set(BYTEKEY, []byte("A"), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get(BYTEKEY)
	if len(res) == 0 {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BYTEKEY))
	}

	if string(res) != "A" {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %v", res, 65))
	}
}

func TestRedis_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getRedisClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestRedis_DeleteRequestInCache(t *testing.T) {
	client, _ := RedisConnectionFactory(tests.MockConfiguration(tests.RedisConfiguration))
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestRedis_Init(t *testing.T) {
	client, _ := RedisConnectionFactory(tests.MockConfiguration(tests.RedisConfiguration))
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Redis provider")
	}
}
