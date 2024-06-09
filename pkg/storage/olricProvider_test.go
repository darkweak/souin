package storage

import (
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/tests"

	"github.com/darkweak/souin/configurationtypes"
)

func getOlricClientAndMatchedURL(key string) (types.Storer, configurationtypes.URL) {
	return GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.OlricConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
			provider, _ := OlricConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

func TestIShouldBeAbleToReadAndWriteDataInOlric(t *testing.T) {
	client, u := getOlricClientAndMatchedURL("Test")
	defer func() {
		_ = client.Reset()
	}()
	_ = client.Set("Test", []byte(BASE_VALUE), u, time.Duration(10)*time.Second)
	time.Sleep(3 * time.Second)
	res := client.Get("Test")
	if BASE_VALUE != string(res) {
		t.Errorf("%s not corresponding to %s", res, BASE_VALUE)
	}
}

func TestOlric_GetRequestInCache(t *testing.T) {
	client, _ := getOlricClientAndMatchedURL(NONEXISTENTKEY)
	defer func() {
		_ = client.Reset()
	}()
	res := client.Get(NONEXISTENTKEY)
	if string(res) != "" {
		t.Errorf("Key %s should not exist", NONEXISTENTKEY)
	}
}

func TestOlric_SetRequestInCache_OneByte(t *testing.T) {
	client, u := getOlricClientAndMatchedURL(BYTEKEY)
	defer func() {
		_ = client.Reset()
	}()
	_ = client.Set(BYTEKEY, []byte{65}, u, time.Duration(20)*time.Second)
}

func TestOlric_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getOlricClientAndMatchedURL(key)
	defer func() {
		_ = client.Reset()
	}()
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestOlric_SetRequestInCache_NoTTL(t *testing.T) {
	client, matchedURL := getOlricClientAndMatchedURL(BYTEKEY)
	defer func() {
		_ = client.Reset()
	}()
	nv := []byte("New value")
	setValueThenVerify(client, BYTEKEY, nv, matchedURL, 0, t)
}

func TestOlric_DeleteRequestInCache(t *testing.T) {
	client, _ := getOlricClientAndMatchedURL(BYTEKEY)
	defer func() {
		_ = client.Reset()
	}()
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		t.Errorf("Key %s should not exist", BYTEKEY)
	}
}

func TestOlric_Init(t *testing.T) {
	client, _ := OlricConnectionFactory(tests.MockConfiguration(tests.OlricConfiguration))
	err := client.Init()
	defer func() {
		_ = client.Reset()
	}()

	if nil != err {
		t.Error("Impossible to init Olric provider")
	}
}
