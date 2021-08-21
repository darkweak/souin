package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/tests"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

const OLRICVALUE = "My first data"

func getOlricClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return tests.GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.OlricConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error) {
			provider, _ := OlricConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

func TestIShouldBeAbleToReadAndWriteDataInOlric(t *testing.T) {
	client, u := getOlricClientAndMatchedURL("Test")
	defer client.Reset()
	client.Set("Test", []byte(OLRICVALUE), u, time.Duration(10)*time.Second)
	time.Sleep(3 * time.Second)
	res := client.Get("Test")
	if OLRICVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, OLRICVALUE))
	}
}

func TestOlric_GetRequestInCache(t *testing.T) {
	client, _ := getOlricClientAndMatchedURL(NONEXISTENTKEY)
	defer client.Reset()
	res := client.Get(NONEXISTENTKEY)
	if string(res) != "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestOlric_SetRequestInCache_OneByte(t *testing.T) {
	client, u := getOlricClientAndMatchedURL(BYTEKEY)
	defer client.Reset()
	client.Set(BYTEKEY, []byte{65}, u, time.Duration(20)*time.Second)
}

func TestOlric_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getOlricClientAndMatchedURL(key)
	defer client.Reset()
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestOlric_SetRequestInCache_NoTTL(t *testing.T) {
	client, matchedURL := getOlricClientAndMatchedURL(BYTEKEY)
	defer client.Reset()
	nv := []byte("New value")
	setValueThenVerify(client, BYTEKEY, nv, matchedURL, 0, t)
}

func TestOlric_DeleteRequestInCache(t *testing.T) {
	client, _ := getOlricClientAndMatchedURL(BYTEKEY)
	defer client.Reset()
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestOlric_Init(t *testing.T) {
	client, _ := OlricConnectionFactory(tests.MockConfiguration(tests.OlricConfiguration))
	err := client.Init()
	defer client.Reset()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Olric provider")
	}
}
