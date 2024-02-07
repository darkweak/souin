package storage

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/tests"

	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

func getNutsClientAndMatchedURL(key string) (types.Storer, configurationtypes.URL) {
	return GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.NutsConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
			provider, _ := NutsConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

func TestNutsConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration(tests.NutsConfiguration)
	r, err := NutsConnectionFactory(c)

	if nil != err {
		errors.GenerateError(t, "Shouldn't have panic")
	}

	if nil == r {
		errors.GenerateError(t, "Nuts should be instanciated")
	}

	if nil == r.(*Nuts).DB {
		errors.GenerateError(t, "Nuts database should be accesible")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInNuts(t *testing.T) {
	client, matchedURL := getNutsClientAndMatchedURL("Test")

	_ = client.Set("Test", []byte(BASE_VALUE), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get("Test")
	if res == nil || len(res) <= 0 {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BASE_VALUE))
	}
	if BASE_VALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", string(res), BASE_VALUE))
	}
}

func TestNuts_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration(tests.NutsConfiguration)
	client, _ := NutsConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestNuts_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getNutsClientAndMatchedURL(BYTEKEY)
	_ = client.Set(BYTEKEY, []byte("A"), matchedURL, time.Duration(20)*time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get(BYTEKEY)
	if len(res) == 0 {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BYTEKEY))
	}

	if string(res) != "A" {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %v", res, 65))
	}
}

func TestNuts_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getNutsClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestNuts_DeleteRequestInCache(t *testing.T) {
	client, _ := NutsConnectionFactory(tests.MockConfiguration(tests.NutsConfiguration))
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestNuts_Init(t *testing.T) {
	client, _ := NutsConnectionFactory(tests.MockConfiguration(tests.NutsConfiguration))
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Nuts provider")
	}
}
