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

const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"

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

func TestInitializeProvider(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	p := InitializeProvider(c)
	err := p.Init()
	if nil != err {
		errors.GenerateError(t, "Init shouldn't crash")
	}
}
