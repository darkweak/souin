package storage

import (
	"testing"

	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/tests"

	"time"

	"github.com/darkweak/souin/configurationtypes"
)

func getOtterClientAndMatchedURL(key string) (types.Storer, configurationtypes.URL) {
	return GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.BaseConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
			provider, _ := OtterConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

// This test ensure that Otter options are override by the Souin configuration
func TestCustomOtterConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration(tests.BadgerConfiguration)
	r, err := OtterConnectionFactory(c)

	if nil != err {
		t.Error("Shouldn't have panic")
	}

	if nil == r {
		t.Error("Otter should be instanciated")
	}
}

func TestOtterConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	r, err := OtterConnectionFactory(c)

	if nil != err {
		t.Error("Shouldn't have panic")
	}

	if nil == r {
		t.Error("Otter should be instanciated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInOtter(t *testing.T) {
	client, matchedURL := getOtterClientAndMatchedURL("Test")

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

func TestOtter_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	client, _ := OtterConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		t.Errorf("Key %s should not exist", NONEXISTENTKEY)
	}
}

func TestOtter_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getOtterClientAndMatchedURL(BYTEKEY)
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

func TestOtter_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getOtterClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestOtter_SetRequestInCache_Negative_TTL(t *testing.T) {
	client, matchedURL := getOtterClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	_ = client.Set(BYTEKEY, nv, matchedURL, -1)
	time.Sleep(1 * time.Second)
	verifyNewValueAfterSet(client, BYTEKEY, []byte{}, t)
}

func TestOtter_DeleteRequestInCache(t *testing.T) {
	client, _ := OtterConnectionFactory(tests.MockConfiguration(tests.BaseConfiguration))
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		t.Errorf("Key %s should not exist", BYTEKEY)
	}
}

func TestOtter_Init(t *testing.T) {
	client, _ := OtterConnectionFactory(tests.MockConfiguration(tests.BaseConfiguration))
	err := client.Init()

	if nil != err {
		t.Error("Impossible to init Otter provider")
	}
}
