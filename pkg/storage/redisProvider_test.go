package storage

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/tests"

	"time"

	"github.com/darkweak/souin/configurationtypes"
)

func getRedisClientAndMatchedURL(key string) (types.Storer, configurationtypes.URL) {
	return GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.RedisConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
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
		t.Error("Shouldn't have panic")
	}

	if nil == r {
		t.Error("Redis should be instanciated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInRedis(t *testing.T) {
	client, matchedURL := getRedisClientAndMatchedURL("Test")

	_ = client.Set("Test", []byte(BASE_VALUE), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get("Test")
	if res == nil || len(res) <= 0 {
		t.Errorf("Key %s should exist", BASE_VALUE)
	}
	if BASE_VALUE != string(res) {
		t.Errorf("%s not corresponding to %s", string(res), BASE_VALUE)
	}
}

func TestRedis_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration(tests.RedisConfiguration)
	client, _ := RedisConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		t.Errorf("Key %s should not exist", NONEXISTENTKEY)
	}
}

func TestRedis_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getRedisClientAndMatchedURL(BYTEKEY)
	_ = client.Set(BYTEKEY, []byte("A"), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get(BYTEKEY)
	if len(res) == 0 {
		t.Errorf("Key %s should exist", BYTEKEY)
	}

	if string(res) != "A" {
		t.Errorf("%s not corresponding to %v", res, 65)
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
		t.Errorf("Key %s should not exist", BYTEKEY)
	}
}

func TestRedis_Init(t *testing.T) {
	client, _ := RedisConnectionFactory(tests.MockConfiguration(tests.RedisConfiguration))
	err := client.Init()

	if nil != err {
		t.Error("Impossible to init Redis provider")
	}
}

func TestRedis_MapKeys(t *testing.T) {
	client, _ := RedisConnectionFactory(tests.MockConfiguration(tests.RedisConfiguration))

	max := 10
	prefix := "MAP_KEYS_PREFIX_"
	m := client.MapKeys(prefix)
	if len(m) != 0 {
		t.Error("The map should be empty")
	}

	for i := 0; i < max; i++ {
		_ = client.Set(fmt.Sprintf("%s%d", prefix, i), []byte(fmt.Sprintf("Hello from %d", i)), configurationtypes.URL{}, time.Second)
	}

	m = client.MapKeys(prefix)
	if len(m) != max {
		t.Errorf("The map should contain %d elements, %d given", max, len(m))
	}

	for k, v := range m {
		if v != fmt.Sprintf("Hello from %s", k) {
			t.Errorf("Expected Hello from %s, %s given", k, v)
		}
	}
}

func TestRedis_DeleteMany(t *testing.T) {
	client, _ := RedisConnectionFactory(tests.MockConfiguration(tests.RedisConfiguration))

	fmt.Println(client.MapKeys(""))
	fmt.Println(len(client.MapKeys("")))
	if len(client.MapKeys("")) != 12 {
		t.Error("The map should contain 12 elements")
	}

	client.DeleteMany("MAP_KEYS_PREFIX_*")
	if len(client.MapKeys("")) != 2 {
		t.Error("The map should contain 2 element")
	}

	client.DeleteMany("*")
	if len(client.MapKeys("")) != 0 {
		t.Error("The map should be empty")
	}
}
