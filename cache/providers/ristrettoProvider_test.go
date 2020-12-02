package providers

import (
	"fmt"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/tests"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"time"
)

const RISTRETTOVALUE = "My first data"
const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"

func getRistrettoClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return tests.GetCacheProviderClientAndMatchedURL(key, func(configurationInterface configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error) {
		provider, _ := RistrettoConnectionFactory(configurationInterface)

		return provider, nil
	})
}

func TestRistrettoConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration()
	r, err := RistrettoConnectionFactory(c)

	if nil != err {
		errors.GenerateError(t, "Shouldn't have panic")
	}

	if nil == r {
		errors.GenerateError(t, "Ristretto should be instanciated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInRistretto(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL("Test")

	client.Set("Test", []byte(RISTRETTOVALUE), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get("Test")
	if res == nil || len(res) <= 0 {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", RISTRETTOVALUE))
	}
	if RISTRETTOVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", string(res), RISTRETTOVALUE))
	}
}

func TestRistretto_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration()
	client, _ := RistrettoConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestRistretto_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(BYTEKEY)
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

func TestRistretto_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getRistrettoClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestRistretto_SetRequestInCache_NoTTL(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	setValueThenVerify(client, BYTEKEY, nv, matchedURL, 0, t)
}

func TestRistretto_SetRequestInCache_NegativeTTL(t *testing.T) {

	client, matchedURL := getRistrettoClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	tests.ValidatePanic(t, func() {
		setValueThenVerify(client, BYTEKEY, nv, matchedURL, time.Duration(-20)*time.Second, t)
	})
}

func TestRistretto_DeleteRequestInCache(t *testing.T) {
	client, _ := RistrettoConnectionFactory(tests.MockConfiguration())
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestRistretto_Init(t *testing.T) {
	client, _ := RistrettoConnectionFactory(tests.MockConfiguration())
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Ristretto provider")
	}
}
