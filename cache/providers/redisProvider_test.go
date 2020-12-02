package providers

import (
	"fmt"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/tests"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

const REDISVALUE = "My first data"

func getRedisClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return tests.GetCacheProviderClientAndMatchedURL(key, func(configurationInterface configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error) {
		provider, _ := RedisConnectionFactory(configurationInterface)

		return provider, nil
	})
}

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client, u := getRedisClientAndMatchedURL("Test")
	client.Set("Test", []byte(REDISVALUE), u, time.Duration(10)*time.Second)
	res := client.Get("Test")
	if REDISVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, REDISVALUE))
	}
}

func TestRedis_GetRequestInCache(t *testing.T) {
	client, _ := getRedisClientAndMatchedURL(NONEXISTENTKEY)
	res := client.Get(NONEXISTENTKEY)
	if string(res) != "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestRedis_SetRequestInCache_OneByte(t *testing.T) {
	client, u := getRedisClientAndMatchedURL(BYTEKEY)
	client.Set(BYTEKEY, []byte{65}, u, time.Duration(20) * time.Second)
}

func TestRedis_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getRedisClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20) * time.Second, t)
}

func TestRedis_SetRequestInCache_NoTTL(t *testing.T) {
	client, matchedURL := getRedisClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	setValueThenVerify(client, BYTEKEY, nv, matchedURL, 0, t)
}

func TestRedis_DeleteRequestInCache(t *testing.T) {
	client, _ := RedisConnectionFactory(tests.MockConfiguration())
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestRedis_Init(t *testing.T) {
	client, _ := RedisConnectionFactory(tests.MockConfiguration())
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Redis provider")
	}
}
