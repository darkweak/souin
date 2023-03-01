package storage

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/tests"

	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

func getEtcdClientAndMatchedURL(key string) (Storer, configurationtypes.URL) {
	return GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.EtcdConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (Storer, error) {
			provider, _ := EtcdConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

func TestEtcdConnectionFactory(t *testing.T) {
	ch := make(chan bool)
	go func() {
		select {
		case <-time.After(3 * time.Second):
			panic("It should not take more than 3 seconds to connect to the etcd instance")
		case <-ch:
		}
	}()

	c := tests.MockConfiguration(tests.EtcdConfiguration)
	r, err := EtcdConnectionFactory(c)
	ch <- true

	if nil != err {
		errors.GenerateError(t, "Shouldn't have panic")
	}

	if nil == r {
		errors.GenerateError(t, "Etcd should be instanciated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInEtcd(t *testing.T) {
	client, matchedURL := getEtcdClientAndMatchedURL("Test")

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

func TestEtcd_GetRequestInCache(t *testing.T) {
	c := tests.MockConfiguration(tests.EtcdConfiguration)
	client, _ := EtcdConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestEtcd_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getEtcdClientAndMatchedURL(BYTEKEY)
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

func TestEtcd_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getEtcdClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestEtcd_DeleteRequestInCache(t *testing.T) {
	client, _ := EtcdConnectionFactory(tests.MockConfiguration(tests.EtcdConfiguration))
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestEtcd_Init(t *testing.T) {
	client, _ := EtcdConnectionFactory(tests.MockConfiguration(tests.EtcdConfiguration))
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Etcd provider")
	}
}
