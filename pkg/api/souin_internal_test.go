package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/darkweak/souin/tests"
	"github.com/darkweak/storages/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEvictMapping_OversizedValue(t *testing.T) {
	memoryStorer, err := storage.Factory(tests.MockConfiguration(tests.BaseConfiguration))
	if err != nil {
		t.Fatalf("Impossible to create the storer, %v given", err)
	}

	oversized := strings.Repeat("a", maxMappingValueSize+1)
	if err := memoryStorer.Set(core.MappingKeyPrefix+"oversized", []byte(oversized), time.Minute); err != nil {
		t.Fatalf("Impossible to store the value, %v given", err)
	}

	expired, _ := proto.Marshal(&core.StorageMapper{Mapping: map[string]*core.KeyIndex{
		"expired-key": {
			FreshTime: timestamppb.New(time.Now().Add(-time.Hour)),
			StaleTime: timestamppb.New(time.Now().Add(-time.Hour)),
			RealKey:   "expired-key",
		},
	}})
	if err := memoryStorer.Set(core.MappingKeyPrefix+"valid", expired, time.Minute); err != nil {
		t.Fatalf("Impossible to store the value, %v given", err)
	}

	EvictMapping(memoryStorer)

	if res := memoryStorer.Get(core.MappingKeyPrefix + "oversized"); len(res) != 0 {
		t.Error("The oversized mapping should be deleted without being decoded")
	}

	if res := memoryStorer.Get(core.MappingKeyPrefix + "valid"); len(res) != 0 {
		t.Error("The fully expired mapping should be deleted")
	}
}

func mockPurgeAPI(t *testing.T) (*SouinAPI, types.Storer) {
	t.Helper()

	config := tests.MockConfiguration(tests.BaseConfiguration)
	memoryStorer, err := storage.Factory(config)
	if err != nil {
		t.Fatalf("Impossible to create the storer, %v given", err)
	}
	core.RegisterStorage(memoryStorer)

	return initializeSouin(config, []types.Storer{memoryStorer}, providers.SurrogateFactory(config, memoryStorer.Name()+"-"+memoryStorer.Uuid())), memoryStorer
}

func storeTaggedResponse(t *testing.T, s *SouinAPI, storer types.Storer, key, tag string) {
	t.Helper()

	if err := storer.SetMultiLevel(key, key, []byte("HTTP/1.1 200 OK\r\n\r\n"), http.Header{}, "", time.Minute, key); err != nil {
		t.Fatalf("Impossible to store the key %s, %v given", key, err)
	}

	res := &http.Response{Header: http.Header{"Surrogate-Key": []string{tag}}}
	if err := s.surrogateStorage.Store(res, key, "/uri-"+tag); err != nil {
		t.Fatalf("Impossible to store the surrogate key %s, %v given", tag, err)
	}
}

func purgeSurrogateKey(s *SouinAPI, tag string) {
	rq := httptest.NewRequest("PURGE", s.GetBasePath(), nil)
	rq.Header.Set("Surrogate-Key", tag)
	s.HandleRequest(httptest.NewRecorder(), rq)
}

func isIndexedUnder(s *SouinAPI, tag, key string) bool {
	return containsKey(s.surrogateStorage.List()[tag], url.QueryEscape(key))
}

func containsKey(value, key string) bool {
	for _, v := range strings.Split(value, ",") {
		if v == key {
			return true
		}
	}

	return false
}

func TestHandleRequest_PurgeSurrogateKeyKeepsOtherTagsIndex(t *testing.T) {
	s, storer := mockPurgeAPI(t)

	r1 := "GET-http-domain.com-/one"
	r2 := "GET-http-domain.com-/two"
	storeTaggedResponse(t, s, storer, r1, "/a/1")
	storeTaggedResponse(t, s, storer, r2, "/a")

	purgeSurrogateKey(s, "/a")

	if len(storer.Get(r2)) != 0 {
		t.Errorf("The response %s tagged /a should be purged", r2)
	}
	if len(storer.Get(r1)) == 0 {
		t.Errorf("The response %s tagged /a/1 should not be purged", r1)
	}
	if !isIndexedUnder(s, "/a/1", r1) {
		t.Errorf("The response %s should still be indexed under /a/1, %+v given", r1, s.surrogateStorage.List())
	}

	purgeSurrogateKey(s, "/a/1")

	if len(storer.Get(r1)) != 0 {
		t.Errorf("The response %s should be purged by its own tag /a/1", r1)
	}
}

func TestHandleRequest_PurgeSurrogateKeyDeletesExactCacheKeys(t *testing.T) {
	cases := map[string]struct{ purged, kept string }{
		"plus": {
			purged: "GET-https-domain.com-/items-application/ld+json",
			kept:   "GET-https-domain.com-/items-application/lddjson",
		},
		"star": {
			purged: "GET-https-domain.com-/items-*/*",
			kept:   "GET-https-domain.com-/items-text/html",
		},
		"dot": {
			purged: "GET-https-domain.com-/items.json",
			kept:   "GET-https-domain.com-/items-json",
		},
		"prefix": {
			purged: "GET-https-domain.com-/items",
			kept:   "GET-https-domain.com-/items/1",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			s, storer := mockPurgeAPI(t)

			storeTaggedResponse(t, s, storer, tc.purged, "purged-tag")
			storeTaggedResponse(t, s, storer, tc.kept, "kept-tag")

			purgeSurrogateKey(s, "purged-tag")

			if len(storer.Get(tc.purged)) != 0 {
				t.Errorf("The response %s should be purged", tc.purged)
			}
			if len(storer.Get(tc.kept)) == 0 {
				t.Errorf("The response %s should not be purged by the key %s", tc.kept, tc.purged)
			}
			if !isIndexedUnder(s, "kept-tag", tc.kept) {
				t.Errorf("The response %s should still be indexed under kept-tag, %+v given", tc.kept, s.surrogateStorage.List())
			}
		})
	}
}

func TestHandleRequest_InvalidateGroupKeepsOtherTagsIndex(t *testing.T) {
	s, storer := mockPurgeAPI(t)

	r1 := "GET-http-domain.com-/one"
	r2 := "GET-http-domain.com-/two"
	storeTaggedResponse(t, s, storer, r1, "/a/1")
	storeTaggedResponse(t, s, storer, r2, "/a")

	rq := httptest.NewRequest(http.MethodPost, s.GetBasePath(), strings.NewReader(`{"type":"group","groups":["/a"],"purge":true}`))
	s.HandleRequest(httptest.NewRecorder(), rq)

	if len(storer.Get(r2)) != 0 {
		t.Errorf("The response %s tagged /a should be invalidated", r2)
	}
	if _, ok := s.surrogateStorage.List()["/a"]; ok {
		t.Errorf("The tag /a should be removed from the surrogate storage, %+v given", s.surrogateStorage.List())
	}
	if !isIndexedUnder(s, "/a/1", r1) {
		t.Errorf("The response %s should still be indexed under /a/1, %+v given", r1, s.surrogateStorage.List())
	}
}

// Mappings written by SetMultiLevel must be reachable by the eviction on the
// default storage, and the mapping it rewrites must stay usable (#885).
func TestEvictMapping_DefaultStorageMultiLevel(t *testing.T) {
	memoryStorer, err := storage.Factory(tests.MockConfiguration(tests.BaseConfiguration))
	if err != nil {
		t.Fatalf("Impossible to create the storer, %v given", err)
	}

	expired := -24 * time.Hour
	_ = memoryStorer.SetMultiLevel("gone", "gone-varied", []byte("body"), http.Header{}, "", expired, "gone-real")
	_ = memoryStorer.SetMultiLevel("partial", "partial-old", []byte("body"), http.Header{"Accept": {"a"}}, "", expired, "partial-old")
	_ = memoryStorer.SetMultiLevel("partial", "partial-new", []byte("body"), http.Header{"Accept": {"b"}}, "", time.Hour, "partial-new")

	EvictMapping(memoryStorer)

	if _, ok := memoryStorer.MapKeys(core.MappingKeyPrefix)["gone"]; ok {
		t.Error("the fully expired mapping should be deleted")
	}
	mapping, ok := memoryStorer.MapKeys(core.MappingKeyPrefix)["partial"]
	if !ok {
		t.Fatal("the mapping with a fresh entry should be kept")
	}
	if strings.Contains(mapping, "partial-old") || !strings.Contains(mapping, "partial-new") {
		t.Errorf("only the expired entry should be evicted, got %q", mapping)
	}

	// The eviction rewrote the mapping, storing and reading it again must work.
	if err := memoryStorer.SetMultiLevel("partial", "partial-other", []byte("body"), http.Header{"Accept": {"c"}}, "", time.Hour, "partial-other"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if keys := memoryStorer.ListKeys(); !slices.Contains(keys, "partial-new") || !slices.Contains(keys, "partial-other") {
		t.Errorf("expected both fresh keys to be listed, got %v", keys)
	}
}

func storeFreshResponse(t *testing.T, s *SouinAPI, storer types.Storer, key, tag string) {
	t.Helper()

	expiry := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	stored := []byte("HTTP/1.1 200 OK\r\nContent-Length: 4\r\nX-Souin-Stored-Expiry: " + expiry + "\r\n\r\nbody")
	if err := storer.SetMultiLevel(key, key, stored, http.Header{}, "", time.Hour, key); err != nil {
		t.Fatalf("Impossible to store the key %s, %v given", key, err)
	}

	res := &http.Response{Header: http.Header{"Surrogate-Key": []string{tag}}}
	if err := s.surrogateStorage.Store(res, key, "/uri-"+tag); err != nil {
		t.Fatalf("Impossible to store the surrogate key %s, %v given", tag, err)
	}
}

func softPurgeRequest(s *SouinAPI, target string, headers map[string]string) int {
	rq := httptest.NewRequest("PURGE", target, nil)
	rq.Header.Set(SoftPurgeModeHeader, "soft")
	for k, v := range headers {
		rq.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.HandleRequest(rec, rq)

	return rec.Code
}

// isStale reads the stored response back the way a request would.
func isStale(t *testing.T, storer types.Storer, key string) bool {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "http://domain.com/", nil)
	fresh, stale := storer.GetMultiLevel(key, req, rfc.ParseRequest(req))
	res := fresh
	if res == nil {
		res = stale
	}
	if res == nil {
		t.Fatalf("The response %s should still be stored", key)
	}

	return !rfc.IsStoredFresh(res.Header, time.Now())
}

func TestHandleRequest_SoftPurgeSurrogateKeyKeepsResponseAndIndex(t *testing.T) {
	s, storer := mockPurgeAPI(t)

	soft := "GET-http-domain.com-/soft"
	other := "GET-http-domain.com-/other"
	storeFreshResponse(t, s, storer, soft, "soft-tag")
	storeFreshResponse(t, s, storer, other, "other-tag")

	if code := softPurgeRequest(s, s.GetBasePath(), map[string]string{"Surrogate-Key": "soft-tag"}); code != http.StatusNoContent {
		t.Fatalf("Unexpected status code %d", code)
	}

	if !isStale(t, storer, soft) {
		t.Errorf("The response %s should be stale after the soft purge", soft)
	}
	if isStale(t, storer, other) {
		t.Errorf("The response %s was not purged and should stay fresh", other)
	}
	if !isIndexedUnder(s, "soft-tag", soft) {
		t.Errorf("The soft-purged response should stay indexed under its tag, %+v given", s.surrogateStorage.List())
	}
	if len(storedMapping(storer, soft)) == 0 {
		t.Error("The mapping of a soft-purged response must be left alone")
	}

	// Nothing but the stored response changed: no marker key next to it.
	for _, k := range storer.ListKeys() {
		if k != soft && k != other {
			t.Errorf("Unexpected key %s, soft purge must not add keys", k)
		}
	}
}

func TestHandleRequest_SoftPurgeKeepsTheRetention(t *testing.T) {
	s, storer := mockPurgeAPI(t)

	key := "GET-http-domain.com-/soft"
	storeFreshResponse(t, s, storer, key, "soft-tag")
	softPurgeRequest(s, s.GetBasePath(), map[string]string{"Surrogate-Key": "soft-tag"})

	// A second soft purge is a no-op rather than a way to lose the entry.
	softPurgeRequest(s, s.GetBasePath(), map[string]string{"Surrogate-Key": "soft-tag"})
	if len(storer.Get(key)) == 0 {
		t.Error("The soft-purged response must outlive the purge")
	}
}

func TestHandleRequest_SoftPurgeRegexpMatchesKeysAsRegexp(t *testing.T) {
	s, storer := mockPurgeAPI(t)

	a := "GET-http-domain.com-/items/1"
	b := "GET-http-domain.com-/items/2"
	c := "GET-http-domain.com-/other"
	storeFreshResponse(t, s, storer, a, "a")
	storeFreshResponse(t, s, storer, b, "b")
	storeFreshResponse(t, s, storer, c, "c")

	if code := softPurgeRequest(s, s.GetBasePath()+"/.*/items/.*", nil); code != http.StatusNoContent {
		t.Fatalf("Unexpected status code %d", code)
	}

	if !isStale(t, storer, a) || !isStale(t, storer, b) {
		t.Error("Both responses matching the regexp should be stale")
	}
	if isStale(t, storer, c) {
		t.Error("The response that does not match should stay fresh")
	}
}

func TestHandleRequest_SoftPurgeFlushIsRefused(t *testing.T) {
	s, storer := mockPurgeAPI(t)

	key := "GET-http-domain.com-/soft"
	storeFreshResponse(t, s, storer, key, "soft-tag")

	if code := softPurgeRequest(s, s.GetBasePath()+"/flush", nil); code != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", code)
	}
	if len(storer.Get(key)) == 0 {
		t.Error("A refused soft flush must not drop anything")
	}
}

func TestBulkDelete_WithoutPurgeSoftPurges(t *testing.T) {
	s, storer := mockPurgeAPI(t)

	key := "GET-http-domain.com-/soft"
	storeFreshResponse(t, s, storer, key, "soft-tag")

	s.BulkDelete(key, false)

	if !isStale(t, storer, key) {
		t.Error("BulkDelete without purge should keep the response and make it stale")
	}
}

func mockPurgeAPIWithStrategy(t *testing.T, strategy string) (*SouinAPI, types.Storer) {
	t.Helper()

	config := tests.MockConfiguration(tests.BaseConfiguration)
	config.DefaultCache.CDN.Strategy = strategy
	memoryStorer, err := storage.Factory(config)
	if err != nil {
		t.Fatalf("Impossible to create the storer, %v given", err)
	}
	core.RegisterStorage(memoryStorer)

	return initializeSouin(config, []types.Storer{memoryStorer}, providers.SurrogateFactory(config, memoryStorer.Name()+"-"+memoryStorer.Uuid())), memoryStorer
}

func TestHandleRequest_CDNStrategySoftSoftPurgesSurrogateKeys(t *testing.T) {
	s, storer := mockPurgeAPIWithStrategy(t, "soft")

	key := "GET-http-domain.com-/strategy"
	storeFreshResponse(t, s, storer, key, "strategy-tag")
	purgeSurrogateKey(s, "strategy-tag")

	if !isStale(t, storer, key) {
		t.Error("cdn.strategy soft should make the purge by surrogate key a soft one")
	}
	if !isIndexedUnder(s, "strategy-tag", key) {
		t.Error("the tag index should be kept")
	}
}

func TestHandleRequest_PurgeModeHardOverridesCDNStrategy(t *testing.T) {
	s, storer := mockPurgeAPIWithStrategy(t, "soft")

	key := "GET-http-domain.com-/strategy"
	storeFreshResponse(t, s, storer, key, "strategy-tag")

	rq := httptest.NewRequest("PURGE", s.GetBasePath(), nil)
	rq.Header.Set("Surrogate-Key", "strategy-tag")
	rq.Header.Set(SoftPurgeModeHeader, "hard")
	s.HandleRequest(httptest.NewRecorder(), rq)

	if len(storer.Get(key)) != 0 {
		t.Error("an explicit hard purge must delete the response whatever the strategy")
	}
}

// The strategy is about purging by tag: the regexp purge and the flush keep
// their meaning, so nobody loses a flush because of a CDN setting.
func TestHandleRequest_CDNStrategySoftLeavesRegexpAndFlushHard(t *testing.T) {
	s, storer := mockPurgeAPIWithStrategy(t, "soft")

	a := "GET-http-domain.com-/regexp"
	b := "GET-http-domain.com-/flush"
	storeFreshResponse(t, s, storer, a, "a")
	storeFreshResponse(t, s, storer, b, "b")

	s.HandleRequest(httptest.NewRecorder(), httptest.NewRequest("PURGE", s.GetBasePath()+"/.*/regexp$", nil))
	if len(storer.Get(a)) != 0 {
		t.Error("the regexp purge without a mode must stay a hard purge")
	}

	rec := httptest.NewRecorder()
	s.HandleRequest(rec, httptest.NewRequest("PURGE", s.GetBasePath()+"/flush", nil))
	if rec.Code != http.StatusNoContent || len(storer.Get(b)) != 0 {
		t.Errorf("the flush must still flush, got %d", rec.Code)
	}
}

func TestHandleRequest_DefaultStrategyStaysHard(t *testing.T) {
	for _, strategy := range []string{"", "hard"} {
		s, storer := mockPurgeAPIWithStrategy(t, strategy)

		key := "GET-http-domain.com-/strategy"
		storeFreshResponse(t, s, storer, key, "strategy-tag")
		purgeSurrogateKey(s, "strategy-tag")

		if len(storer.Get(key)) != 0 {
			t.Errorf("strategy %q: the purge by surrogate key must delete the response", strategy)
		}
	}
}
