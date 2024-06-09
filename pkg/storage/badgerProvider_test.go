package storage

import (
	"testing"

	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/tests"

	"time"

	"github.com/darkweak/souin/configurationtypes"
)

func getBadgerClientAndMatchedURL(key string) (types.Storer, configurationtypes.URL) {
	return GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.BaseConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.Storer, error) {
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
		t.Error("Shouldn't have panic")
	}

	if nil == r {
		t.Error("Badger should be instanciated")
	}
}

func TestBadgerConnectionFactory(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	r, err := BadgerConnectionFactory(c)

	if nil != err {
		t.Error("Shouldn't have panic")
	}

	if nil == r {
		t.Error("Badger should be instanciated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInBadger(t *testing.T) {
	client, matchedURL := getBadgerClientAndMatchedURL("Test")

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

func TestBadger_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	client, _ := BadgerConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		t.Errorf("Key %s should not exist", NONEXISTENTKEY)
	}
}

func TestBadger_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getBadgerClientAndMatchedURL(BYTEKEY)
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

func TestBadger_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getBadgerClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestBadger_SetRequestInCache_Negative_TTL(t *testing.T) {
	client, matchedURL := getBadgerClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	_ = client.Set(BYTEKEY, nv, matchedURL, -1)
	time.Sleep(1 * time.Second)
	verifyNewValueAfterSet(client, BYTEKEY, []byte{}, t)
}

func TestBadger_DeleteRequestInCache(t *testing.T) {
	client, _ := BadgerConnectionFactory(tests.MockConfiguration(tests.BaseConfiguration))
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		t.Errorf("Key %s should not exist", BYTEKEY)
	}
}

func TestBadger_Init(t *testing.T) {
	client, _ := BadgerConnectionFactory(tests.MockConfiguration(tests.BaseConfiguration))
	err := client.Init()

	if nil != err {
		t.Error("Impossible to init Badger provider")
	}
}
