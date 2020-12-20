package providers

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"time"
)

const RISTRETTOVALUE = "My first data"
const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"
const DELETABLEKEY = "MyDeletableKey"

func getRistrettoClientAndMatchedURL(key string) (*Ristretto, configurationtypes.URL) {
	config := MockConfiguration()
	client, _ := RistrettoConnectionFactory(config)
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configurationtypes.URL{
		TTL:     config.GetDefaultCache().TTL,
		Headers: config.GetDefaultCache().Headers,
	}
	if "" != regexpURL {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return client, matchedURL
}

func TestRistrettoConnectionFactory(t *testing.T) {
	c := MockConfiguration()
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

	client.Set("Test", []byte(RISTRETTOVALUE), matchedURL, time.Duration(20) * time.Second)
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
	c := MockConfiguration()
	client, _ := RistrettoConnectionFactory(c)
	res := client.Get(NONEXISTENTKEY)
	if 0 < len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestRistretto_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(BYTEKEY)
	client.Set(BYTEKEY, []byte("A"), matchedURL, time.Duration(20) * time.Second)
	time.Sleep(1 * time.Second)

	res := client.Get(BYTEKEY)
	if 0 >= len(res) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BYTEKEY))
	}

	if string(res) != "A" {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %v", res, 65))
	}
}

func TestRistretto_SetRequestInCache_MultipleKeys(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(DELETABLEKEY)

	for i := 0; i < 10; i++ {
		client.Set(fmt.Sprintf("%s%v", DELETABLEKEY, i), []byte{65}, matchedURL, time.Duration(20) * time.Second)
	}
}

func TestRistretto_SetRequestInCache(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL("MyEmptyKey")
	client.Set("MyEmptyKey", []byte("Hello world"), matchedURL, time.Duration(20) * time.Second)
}

func TestRistretto_SetRequestInCache_Empty(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL("MyEmptyKey")
	client.Set("MyEmptyKey", []byte{}, matchedURL, time.Duration(20) * time.Second)
}

func TestRistretto_SetRequestInCache_VeryLong(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL("MyVeryLongKey")
	client.Set("MyVeryLongKey", make([]byte, 100000000), matchedURL, time.Duration(20) * time.Second)
}

func TestRistretto_SetRequestInCache_ExistingKey(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(BYTEKEY)

	for i := 0; i < 10; i++ {
		client.Set(BYTEKEY, []byte("New value"), matchedURL, time.Duration(20) * time.Second)
	}
}

func TestRistretto_DeleteRequestInCache(t *testing.T) {
	client, _ := RistrettoConnectionFactory(MockConfiguration())
	client.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if 0 < len(client.Get(BYTEKEY)) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestRistretto_Init(t *testing.T) {
	client, _ := RistrettoConnectionFactory(MockConfiguration())
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to init Ristretto provider")
	}
}
