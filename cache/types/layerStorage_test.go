package types

import (
	"fmt"
	"testing"

	"github.com/darkweak/souin/errors"
	"time"
)

const LAYEREDKEY = "LayeredKey"
const BYTEKEY = "MyByteKey"
const NONEXISTENTKEY = "NonexistentKey"

func TestInitializeCoalescingLayerStorage(t *testing.T) {
	r := InitializeCoalescingLayerStorage()

	if nil == r || nil == r.Cache {
		errors.GenerateError(t, "Ristretto should be instantiated")
	}
}

func TestIShouldBeAbleToReadAndWriteDataInTheLayerStorage(t *testing.T) {
	store := InitializeCoalescingLayerStorage()

	store.Set(LAYEREDKEY)
	time.Sleep(1 * time.Second)

	if store.Exists(LAYEREDKEY) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should exist", LAYEREDKEY))
	}
}

func TestLayerStorage_GetRequestInTheLayerStorage(t *testing.T) {
	store := InitializeCoalescingLayerStorage()
	if !store.Exists(NONEXISTENTKEY) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", NONEXISTENTKEY))
	}
}

func TestLayerStorage_DeleteRequestInCache(t *testing.T) {
	store := InitializeCoalescingLayerStorage()
	store.Delete(BYTEKEY)
	time.Sleep(1 * time.Second)
	if !store.Exists(BYTEKEY) {
		errors.GenerateError(t, fmt.Sprintf("Key %s should not exist", BYTEKEY))
	}
}
