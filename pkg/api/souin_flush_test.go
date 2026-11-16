package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/darkweak/souin/tests"
	"github.com/darkweak/storages/core"
)

// closingStorer behaves like the redis, go-redis, etcd, olric, nuts and badger
// storages: Reset closes the client, and every write after that fails.
type closingStorer struct {
	types.Storer
	closed bool
}

func (c *closingStorer) Reset() error {
	c.closed = true

	return nil
}

func (c *closingStorer) Set(key string, value []byte, d time.Duration) error {
	if c.closed {
		return errors.New("client is closing")
	}

	return c.Storer.Set(key, value, d)
}

func (c *closingStorer) SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, d time.Duration, realKey string) error {
	if c.closed {
		return errors.New("client is closing")
	}

	return c.Storer.SetMultiLevel(baseKey, variedKey, value, variedHeaders, etag, d, realKey)
}

func TestHandleRequest_FlushKeepsTheStorerOpen(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	inner, err := storage.Factory(config)
	if err != nil {
		t.Fatalf("Impossible to create the storer, %v given", err)
	}

	cache := &closingStorer{Storer: inner}
	core.RegisterStorage(cache)

	s := initializeSouin(config, []types.Storer{cache}, providers.SurrogateFactory(config, cache.Name()+"-"+cache.Uuid()))

	storeTaggedResponse(t, s, cache, "GET-https-example.com-/a", "tag-a")

	if len(s.surrogateStorage.List()) == 0 {
		t.Fatal("The surrogate key should be indexed before the flush")
	}

	s.HandleRequest(httptest.NewRecorder(), httptest.NewRequest("PURGE", s.GetBasePath()+"/flush", nil))

	if cache.closed {
		t.Fatal("The flush must not close the storer shared by the cache and the surrogate storage")
	}

	if len(cache.Get("GET-https-example.com-/a")) != 0 {
		t.Error("The flush should delete the cached response")
	}

	if got := s.surrogateStorage.List(); len(got) != 0 {
		t.Errorf("The flush should delete the surrogate keys, %v left", got)
	}

	storeTaggedResponse(t, s, cache, "GET-https-example.com-/b", "tag-b")

	if !isIndexedUnder(s, "tag-b", "GET-https-example.com-/b") {
		t.Error("The surrogate storage should keep indexing after a flush")
	}
}
