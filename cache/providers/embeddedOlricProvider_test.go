package providers

import (
	"fmt"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"testing"
	"time"
)

const EMBEDDEDOLRICVALUE = "My first data"

const olricConfiguration string = `
olricd:
  bindAddr: "0.0.0.0"
  bindPort: 3320
  serializer: "msgpack"
  keepAlivePeriod: "300s"
  bootstrapTimeout: "5s"
  partitionCount:  271
  replicaCount: 2
  writeQuorum: 1
  readQuorum: 1
  readRepair: false
  replicationMode: 1 # sync mode. for async, set 1
  tableSize: 1048576 # 1MB in bytes
  memberCountQuorum: 1

client:
  dialTimeout: "-1s"
  readTimeout: "3s"
  writeTimeout: "3s"
  keepAlive: "15s"
  minConn: 1
  maxConn: 100

logging:
  verbosity: 6
  level: "DEBUG"
  output: "stderr"

memberlist:
  environment: "local"
  bindAddr: "0.0.0.0"
  bindPort: 3322
  enableCompression: false
  joinRetryInterval: "1s"
  maxJoinAttempts: 10
`

func getEmbeddedOlricClientAndMatchedURL(key string) (types.AbstractProviderInterface, configurationtypes.URL) {
	return tests.GetCacheProviderClientAndMatchedURL(
		key,
		func() configurationtypes.AbstractConfigurationInterface {
			return tests.MockConfiguration(tests.EmbeddedOlricConfiguration)
		},
		func(config configurationtypes.AbstractConfigurationInterface) (types.AbstractProviderInterface, error) {
			provider, _ := EmbeddedOlricConnectionFactory(config)
			_ = provider.Init()

			return provider, nil
		},
	)
}

func TestIShouldBeAbleToReadAndWriteDataInEmbeddedOlric(t *testing.T) {
	client, u := getEmbeddedOlricClientAndMatchedURL("Test")
	defer client.Reset()
	client.Set("Test", []byte(EMBEDDEDOLRICVALUE), u, time.Duration(10)*time.Second)
	time.Sleep(3 * time.Second)
	res := client.Get("Test")
	if EMBEDDEDOLRICVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", res, EMBEDDEDOLRICVALUE))
	}
}

func TestEmbeddedOlric_GetRequestInCache(t *testing.T) {
	client, _ := getEmbeddedOlricClientAndMatchedURL(NONEXISTENTKEY)
	defer client.Reset()
	res := client.Get(NONEXISTENTKEY)
	if string(res) != "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestEmbeddedOlric_SetRequestInCache_OneByte(t *testing.T) {
	client, u := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	defer client.Reset()
	client.Set(BYTEKEY, []byte{65}, u, time.Duration(20)*time.Second)
}

func TestEmbeddedOlric_SetRequestInCache_TTL(t *testing.T) {
	key := "MyEmptyKey"
	client, matchedURL := getEmbeddedOlricClientAndMatchedURL(key)
	defer client.Reset()
	nv := []byte("Hello world")
	setValueThenVerify(client, key, nv, matchedURL, time.Duration(20)*time.Second, t)
}

func TestEmbeddedOlric_SetRequestInCache_NoTTL(t *testing.T) {
	client, matchedURL := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	defer client.Reset()
	nv := []byte("New value")
	setValueThenVerify(client, BYTEKEY, nv, matchedURL, 0, t)
}

func TestEmbeddedOlric_DeleteRequestInCache(t *testing.T) {
	client, _ := getEmbeddedOlricClientAndMatchedURL(BYTEKEY)
	defer client.Reset()
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestEmbeddedOlric_Init(t *testing.T) {
	client, _ := EmbeddedOlricConnectionFactory(tests.MockConfiguration(tests.EmbeddedOlricConfiguration))
	err := client.Init()
	defer client.Reset()

	if nil != err {
		errors.GenerateError(t, "Impossible to init EmbeddedOlric provider")
	}
}
