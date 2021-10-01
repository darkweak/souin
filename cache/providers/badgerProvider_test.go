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

const BADGERVALUE = "My first data"
const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"

func getBadgerClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return tests.GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.BaseConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error) {
			provider, _ := BadgerConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

// This test ensure that Badger options are override by the Souin configuration
func TestCustomBadgerConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration(tests.BadgerConfiguration)
	r, err := BadgerConnectionFactory(c)

	if nil != err {
		errors.GenerateError(t, "Shouldn't have panic")
	}

	if nil == r {
		errors.GenerateError(t, "Badger should be instanciated")
	}
}

func TestBadgerConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	r, err := BadgerConnectionFactory(c)

	if nil != err {
		errors.GenerateError(t, "Shouldn't have panic")
	}

	if nil == r {
		errors.GenerateError(t, "Badger should be instanciated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInBadger(t *testing.T) {
	client, matchedURL := getBadgerClientAndMatchedURL("Test")

	client.Set("Test", []byte(BADGERVALUE), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get("Test")
	if res == nil || len(res) <= 0 {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BADGERVALUE))
	}
	if BADGERVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", string(res), BADGERVALUE))
	}
}

func TestBadger_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	client, _ := BadgerConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestBadger_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getBadgerClientAndMatchedURL(BYTEKEY)
	client.Set(BYTEKEY, []byte("A"), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get(BYTEKEY)
	if 0 == len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BYTEKEY))
	}

	if string(res) != "A" {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %v", res, 65))
	}
}

func verifyNewValueAfterSet(client types.AbstractProviderInterface, key string, value []byte, t *testing.T) {
	newValue := client.Get(key)

	if len(newValue) != len(value) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should be equals to %s, %s provided", key, value, newValue))
	}
}

func setValueThenVerify(client types.AbstractProviderInterface, key string, value []byte, matchedURL configurationtypes.URL, ttl time.Duration, t *testing.T) {
	client.Set(key, value, matchedURL, ttl)
	time.Sleep(1 * time.Second)
	verifyNewValueAfterSet(client, key, value, t)
}

func TestBadger_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getBadgerClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestBadger_SetRequestInCache_Negative_TTL(t *testing.T) {
	client, matchedURL := getBadgerClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	client.Set(BYTEKEY, nv, matchedURL, -1)
	time.Sleep(1 * time.Second)
	verifyNewValueAfterSet(client, BYTEKEY, []byte{}, t)
}

func TestBadger_DeleteRequestInCache(t *testing.T) {
	client, _ := BadgerConnectionFactory(tests.MockConfiguration(tests.BaseConfiguration))
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestBadger_Init(t *testing.T) {
	client, _ := BadgerConnectionFactory(tests.MockConfiguration(tests.BaseConfiguration))
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Badger provider")
	}
}
