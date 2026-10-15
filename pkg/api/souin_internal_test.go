package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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
