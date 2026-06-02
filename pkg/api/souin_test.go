package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/darkweak/souin/tests"
	"github.com/darkweak/storages/core"
	"google.golang.org/protobuf/proto"
)

func newTestSouinAPI(t *testing.T) (*SouinAPI, types.Storer) {
	t.Helper()

	memoryStorer, _ := storage.Factory(tests.MockConfiguration(tests.BaseConfiguration))
	core.RegisterStorage(memoryStorer)

	cfg := tests.MockConfiguration(tests.BaseConfiguration)

	return initializeSouin(cfg, []types.Storer{memoryStorer}, nil), memoryStorer
}

func TestIsSoftPurgeRequest(t *testing.T) {
	reqWithHeader, _ := http.NewRequest("PURGE", "http://example.com/souin-api/souin", nil)
	reqWithHeader.Header.Set(SoftPurgeModeHeader, "soft")
	if !IsSoftPurgeRequest(reqWithHeader) {
		t.Fatal("request with soft purge header should be detected as soft purge")
	}

	reqWithQuery, _ := http.NewRequest("PURGE", "http://example.com/souin-api/souin?mode=soft", nil)
	if IsSoftPurgeRequest(reqWithQuery) {
		t.Fatal("request with only soft purge query parameter should not be detected as soft purge")
	}

	reqHard, _ := http.NewRequest("PURGE", "http://example.com/souin-api/souin", nil)
	reqHard.Header.Set(SoftPurgeModeHeader, "hard")
	if IsSoftPurgeRequest(reqHard) {
		t.Fatal("request with hard purge mode should not be detected as soft purge")
	}
}

func TestBulkDeleteSoftPurgePreservesEntryAndMarksMappingStale(t *testing.T) {
	api, storer := newTestSouinAPI(t)
	cacheKey := "GET-http-example.com-/resource"
	mappingKey := core.MappingKeyPrefix + cacheKey

	if err := storer.Set(cacheKey, []byte("payload"), types.OneYearDuration); err != nil {
		t.Fatalf("unable to store cache entry: %v", err)
	}

	mapping, err := core.MappingUpdater(
		cacheKey,
		nil,
		api.logger,
		time.Now(),
		time.Now().Add(time.Minute),
		time.Now().Add(2*time.Minute),
		nil,
		"",
		cacheKey,
	)
	if err != nil {
		t.Fatalf("unable to create mapping: %v", err)
	}

	if err = storer.Set(mappingKey, mapping, types.OneYearDuration); err != nil {
		t.Fatalf("unable to store mapping: %v", err)
	}

	api.BulkDelete(cacheKey, false)

	if got := storer.Get(cacheKey); len(got) == 0 {
		t.Fatal("soft purge should preserve the stored cache entry")
	}

	if got := storer.Get(SoftPurgeMarkerKey(cacheKey)); len(got) == 0 {
		t.Fatal("soft purge should create a soft purge marker")
	}

	rawMapping := storer.Get(mappingKey)
	if len(rawMapping) == 0 {
		t.Fatal("soft purge should preserve the mapping entry")
	}

	decoded := &core.StorageMapper{}
	if err = proto.Unmarshal(rawMapping, decoded); err != nil {
		t.Fatalf("unable to decode mapping: %v", err)
	}

	item := decoded.GetMapping()[cacheKey]
	if item == nil {
		t.Fatal("soft purge should preserve the cache key mapping")
	}

	if item.GetFreshTime().AsTime().After(time.Now()) {
		t.Fatal("soft purge should mark the mapping as no longer fresh")
	}
}

func TestBulkDeleteHardPurgeRemovesMarkerMappingAndEntry(t *testing.T) {
	api, storer := newTestSouinAPI(t)
	cacheKey := "GET-http-example.com-/resource"

	_ = storer.Set(cacheKey, []byte("payload"), types.OneYearDuration)
	_ = storer.Set(core.MappingKeyPrefix+cacheKey, []byte("mapping"), types.OneYearDuration)
	_ = storer.Set(SoftPurgeMarkerKey(cacheKey), []byte("marker"), types.OneYearDuration)

	api.BulkDelete(cacheKey, true)

	if got := storer.Get(cacheKey); len(got) != 0 {
		t.Fatal("hard purge should remove the stored cache entry")
	}

	if got := storer.Get(core.MappingKeyPrefix + cacheKey); len(got) != 0 {
		t.Fatal("hard purge should remove the mapping entry")
	}

	if got := storer.Get(SoftPurgeMarkerKey(cacheKey)); len(got) != 0 {
		t.Fatal("hard purge should remove the soft purge marker")
	}
}

func TestSoftPurgeFlushIsRejected(t *testing.T) {
	api, _ := newTestSouinAPI(t)
	api.basePath = "/souin"

	req := httptest.NewRequest("PURGE", "/souin/flush", nil)
	req.Header.Set(SoftPurgeModeHeader, "soft")
	res := httptest.NewRecorder()

	api.HandleRequest(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected soft purge flush to return %d, got %d", http.StatusBadRequest, res.Code)
	}

	if body := res.Body.String(); body != "soft purge is not supported for flush" {
		t.Fatalf("unexpected soft purge flush response body %q", body)
	}
}

func TestFlushClearsSharedSurrogateStorageWithoutResettingStorer(t *testing.T) {
	cfg := tests.MockConfiguration(tests.BaseConfiguration)
	storer, _ := storage.Factory(cfg)
	core.RegisterStorage(storer)

	surrogateStorage := providers.SurrogateFactory(cfg, storer.Name())
	api := initializeSouin(cfg, []types.Storer{storer}, surrogateStorage)
	api.basePath = "/souin"

	cacheKey := "GET-http-example.com-/"
	if err := storer.Set(cacheKey, []byte("payload"), types.OneYearDuration); err != nil {
		t.Fatalf("unable to store cache entry: %v", err)
	}
	if err := storer.Set("SURROGATE_blog-1-home", []byte(","+cacheKey), types.OneYearDuration); err != nil {
		t.Fatalf("unable to store surrogate entry: %v", err)
	}

	req := httptest.NewRequest("PURGE", "/souin/flush", nil)
	res := httptest.NewRecorder()
	api.HandleRequest(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected flush to return %d, got %d", http.StatusNoContent, res.Code)
	}

	if got := storer.Get(cacheKey); len(got) != 0 {
		t.Fatal("flush should remove cached entries")
	}

	if got := storer.Get("SURROGATE_blog-1-home"); len(got) != 0 {
		t.Fatal("flush should remove surrogate entries")
	}

	if err := storer.Set("still-open", []byte("ok"), time.Minute); err != nil {
		t.Fatalf("flush should not reset the shared storage: %v", err)
	}
}
