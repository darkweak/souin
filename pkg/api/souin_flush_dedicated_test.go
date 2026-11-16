package api

import (
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/darkweak/souin/tests"
	"github.com/darkweak/storages/core"
)

// idStorer gives a storer its own registration key, so the surrogate storage
// can use a different storer than the cache.
type idStorer struct {
	*closingStorer
	id string
}

func (s *idStorer) Uuid() string { return s.id }

// With the surrogate keys on their own storer, the flush's DeleteMany on the
// cache storers never reaches them: only Clear does.
func TestHandleRequest_FlushClearsADedicatedSurrogateStorer(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)

	cacheInner, err := storage.Factory(config)
	if err != nil {
		t.Fatalf("Impossible to create the cache storer, %v given", err)
	}

	surrogateInner, err := storage.Factory(config)
	if err != nil {
		t.Fatalf("Impossible to create the surrogate storer, %v given", err)
	}

	cache := &closingStorer{Storer: cacheInner}
	surrogateStorer := &idStorer{closingStorer: &closingStorer{Storer: surrogateInner}, id: "dedicated-surrogate"}
	core.RegisterStorage(surrogateStorer)

	s := initializeSouin(config, []types.Storer{cache}, providers.SurrogateFactory(config, surrogateStorer.Name()+"-"+surrogateStorer.Uuid()))

	storeTaggedResponse(t, s, cache, "GET-https-example.com-/a", "tag-a")

	if !isIndexedUnder(s, "tag-a", "GET-https-example.com-/a") {
		t.Fatal("The surrogate key should be indexed on the dedicated storer before the flush")
	}

	if len(surrogateInner.MapKeys("SURROGATE_")) == 0 {
		t.Fatal("The surrogate key should live on the dedicated storer, not on the cache storer")
	}

	s.HandleRequest(httptest.NewRecorder(), httptest.NewRequest("PURGE", s.GetBasePath()+"/flush", nil))

	if surrogateStorer.closed || cache.closed {
		t.Fatal("The flush must not close either storer")
	}

	if got := s.surrogateStorage.List(); len(got) != 0 {
		t.Errorf("The flush should delete the surrogate keys on the dedicated storer, %v left", got)
	}

	storeTaggedResponse(t, s, cache, "GET-https-example.com-/b", "tag-b")

	if !isIndexedUnder(s, "tag-b", "GET-https-example.com-/b") {
		t.Error("The dedicated surrogate storer should keep indexing after a flush")
	}
}
