package providers

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/configuration"
)

const MEMORYVALUE = "My first data"
const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"

func TestIShouldBeAbleToReadAndWriteDataInMemory(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	err := client.Set("Test", []byte(MEMORYVALUE))
	if err != nil {
		errors.GenerateError(t, "Impossible to set redis variable")
	}
	res, err := client.Get( "Test")
	if err != nil {
		errors.GenerateError(t, "Retrieving data from redis")
	}
	if MEMORYVALUE != string(res) {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %s", string(res), MEMORYVALUE))
	}
}

func TestMemory_GetRequestInCache(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	res := client.GetRequestInCache(NONEXISTENTKEY)
	if res.Response != "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestMemory_GetSetRequestInCache_OneByte(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	client.SetRequestInCache(BYTEKEY, []byte("A"))

	res := client.GetRequestInCache(BYTEKEY)
	if res.Response == "" {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", BYTEKEY))
	}

	if res.Response != "A" {
		errors.GenerateError(t, fmt.Sprintf("%s not corresponding to %v", res.Response, 65))
	}
}

func TestMemory_SetRequestInCache_MultipleKeys(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())

	for i:= 0; i < 10; i++ {
		client.SetRequestInCache(fmt.Sprintf("%s%v", DELETABLEKEY, i), []byte{65})
	}
}

func TestMemory_SetRequestInCache_Empty(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	client.SetRequestInCache("MyEmptyKey", []byte{})
}

func TestMemory_SetRequestInCache_VeryLong(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	client.SetRequestInCache("MyVeryLongKey", make([]byte, 100000000))
}

func TestMemory_SetRequestInCache_ExistingKey(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())

	for i:= 0; i < 10; i++ {
		client.SetRequestInCache(BYTEKEY, []byte("New value"))
	}
}

func TestMemory_DeleteRequestInCache(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	client.DeleteRequestInCache(BYTEKEY)
	if "" != client.GetRequestInCache(BYTEKEY).Response {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}

func TestMemory_Init(t *testing.T) {
	client := MemoryConnectionFactory(configuration.GetConfig())
	err := client.Init()

	if nil != err {
		errors.GenerateError(t, "Impossible to instantiate Memory provider")
	}
}
