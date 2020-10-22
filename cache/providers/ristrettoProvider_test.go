package providers

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/errors"
	"time"
	"github.com/darkweak/souin/configuration"
)

const RISTRETTOVALUE = "My first data"
const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"
const DELETABLEKEY = "MyDeletableKey"

func getRistrettoClientAndMatchedURL(key string) (*Ristretto, configuration.URL) {
	config := MockConfiguration()
	client := RistrettoConnectionFactory(config)
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configuration.URL{
		TTL:       config.GetDefaultCache().TTL,
		Headers:   config.GetDefaultCache().Headers,
	}
	if "" != regexpURL {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return client, matchedURL
}

func TestIShouldBeAbleToReadAndWriteDataInRistretto(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL("Test")

	client.SetRequestInCache("Test", []byte(RISTRETTOVALUE), matchedURL)
	time.Sleep(1 * time.Second)

	res, b := client.Get( "Test")
	if false == b || nil == res {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", RISTRETTOVALUE))
	}
	if RISTRETTOVALUE != string(res.([]byte)) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", string(res.([]byte)), RISTRETTOVALUE))
	}
}

func TestRistretto_GetRequestInCache(t *testing.T) {
	c := MockConfiguration()
	client := RistrettoConnectionFactory(c)
	res := client.GetRequestInCache(NONEXISTENTKEY)
	if res.Response != "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestRistretto_GetSetRequestInCache_OneByte(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(BYTEKEY)
	client.SetRequestInCache(BYTEKEY, []byte("A"), matchedURL)
	time.Sleep(1 * time.Second)

	res := client.GetRequestInCache(BYTEKEY)
	if res.Response == "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BYTEKEY))
	}

	if res.Response != "A" {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %v", res.Response, 65))
	}
}

func TestRistretto_SetRequestInCache_MultipleKeys(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(DELETABLEKEY)

	for i:= 0; i < 10; i++ {
		client.SetRequestInCache(fmt.Sprintf("%s%v", DELETABLEKEY, i), []byte{65}, matchedURL)
	}
}

func TestRistretto_SetRequestInCache_Empty(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL("MyEmptyKey")
	client.SetRequestInCache("MyEmptyKey", []byte{}, matchedURL)
}

func TestRistretto_SetRequestInCache_VeryLong(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL("MyVeryLongKey")
	client.SetRequestInCache("MyVeryLongKey", make([]byte, 100000000), matchedURL)
}

func TestRistretto_SetRequestInCache_ExistingKey(t *testing.T) {
	client, matchedURL := getRistrettoClientAndMatchedURL(BYTEKEY)

	for i:= 0; i < 10; i++ {
		client.SetRequestInCache(BYTEKEY, []byte("New value"), matchedURL)
	}
}

func TestRistretto_DeleteRequestInCache(t *testing.T) {
	client := RistrettoConnectionFactory(MockConfiguration())
	client.DeleteRequestInCache(BYTEKEY)
	time.Sleep(1 * time.Second)
	if "" != client.GetRequestInCache(BYTEKEY).Response {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestRistretto_Init(t *testing.T) {
	client := RistrettoConnectionFactory(MockConfiguration())
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to instantiate Ristretto provider")
	}
}
