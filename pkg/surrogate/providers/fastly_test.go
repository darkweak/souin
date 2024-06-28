package providers

import (
	"net/http"
	"sync"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/tests"
	"go.uber.org/zap"
)

func mockFastlyProvider() *FastlySurrogateStorage {
	instanciator, _ := storage.NewStorageFromName("badger")
	storer, _ := instanciator(tests.MockConfiguration(tests.BadgerConfiguration))
	fss := &FastlySurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    storer,
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         &sync.Mutex{},
			logger:     zap.NewNop(),
		},
		providerAPIKey: "my_key",
		serviceID:      "123",
		strategy:       "0",
	}

	fss.baseStorage.parent = fss

	return fss
}

func TestFastlySurrogateStorage_Purge(t *testing.T) {
	ap := mockFastlyProvider()
	headerMock := http.Header{}
	headerMock.Set(surrogateKey, baseHeaderValue)

	cacheKeys1, surrogateKeys1 := ap.baseStorage.Purge(headerMock)
	cacheKeys2, surrogateKeys2 := ap.Purge(headerMock)

	if len(cacheKeys1) != len(cacheKeys2) {
		t.Error("The cache keys length should match.")
	}
	if len(surrogateKeys1) != len(surrogateKeys2) {
		t.Error("The surrogate keys length should match.")
	}

	for i, key := range cacheKeys1 {
		if key != cacheKeys2[i] {
			t.Errorf("The cache key %d should match %s, %s given.", i, key, cacheKeys2[i])
		}
	}
	for i, key := range surrogateKeys1 {
		if key != surrogateKeys2[i] {
			t.Errorf("The surrogate key %d should match %s, %s given.", i, key, surrogateKeys2[i])
		}
	}
}
