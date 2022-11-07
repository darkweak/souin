package providers

import (
	"fmt"
	"testing"
	"time"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

const EMBEDDEDOLRICVALUE = "My first data"

func mockEmbeddedConfiguration(c func() string, key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return tests.GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(c)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error) {
			provider, _ := EmbeddedOlricConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

func getEmbeddedOlricClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return mockEmbeddedConfiguration(tests.EmbeddedOlricConfiguration, key)
}

func getEmbeddedOlricWithoutYAML(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return mockEmbeddedConfiguration(tests.EmbeddedOlricPlainConfigurationWithoutAdditionalYAML, key)
}

func TestIShouldBeAbleToReadAndWriteDataInEmbeddedOlric(t *testing.T) {
	client, u := getEmbeddedOlricClientAndMatchedURL("Test")
	_ = client.Set("Test", []byte(EMBEDDEDOLRICVALUE), u, time.Duration(10)*time.Second)
	time.Sleep(3 * time.Second)
	res := client.Get("Test")
	if EMBEDDEDOLRICVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, EMBEDDEDOLRICVALUE))
	}
	_ = client.Reset()
}

func TestIShouldBeAbleToReadAndWriteDataInEmbeddedOlricWithoutYAML(t *testing.T) {
	client, u := getEmbeddedOlricWithoutYAML("Test_without")
	_ = client.Set("Test", []byte(EMBEDDEDOLRICVALUE), u, time.Duration(10)*time.Second)
	time.Sleep(3 * time.Second)
	res := client.Get("Test")
	if EMBEDDEDOLRICVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, EMBEDDEDOLRICVALUE))
	}
	_ = client.Reset()
}

func TestEmbeddedOlric_GetRequestInCache(t *testing.T) {
	client, _ := getEmbeddedOlricClientAndMatchedURL(NONEXISTENTKEY)
	res := client.Get(NONEXISTENTKEY)
	if string(res) != "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
	_ = client.Reset()
}

func TestEmbeddedOlric_SetRequestInCache_OneByte(t *testing.T) {
	client, u := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	_ = client.Set(BYTEKEY, []byte{65}, u, time.Duration(20)*time.Second)
	_ = client.Reset()
}

func TestEmbeddedOlric_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getEmbeddedOlricClientAndMatchedURL(key)
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
	_ = client.Reset()
}

func TestEmbeddedOlric_SetRequestInCache_NoTTL(t *testing.T) {
	client, matchedURL := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	nv := []byte("New value")
	setValueThenVerify(client, BYTEKEY, nv, matchedURL, 0, t)
	_ = client.Reset()
}

func TestEmbeddedOlric_DeleteRequestInCache(t *testing.T) {
	client, _ := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
	_ = client.Reset()
}

func TestEmbeddedOlric_Init(t *testing.T) {
	client, _ := EmbeddedOlricConnectionFactory(tests.MockConfiguration(tests.EmbeddedOlricConfiguration))
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init EmbeddedOlric provider")
	}
	_ = client.Reset()
}
