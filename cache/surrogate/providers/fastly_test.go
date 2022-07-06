package providers

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"go.uber.org/zap"
)

func mockFastlyProvider() *FastlySurrogateStorage {
	fss := &FastlySurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    make(map[string]string),
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
		errors.GenerateError(t, "The cache keys length should match.")
	}
	if len(surrogateKeys1) != len(surrogateKeys2) {
		errors.GenerateError(t, "The surrogate keys length should match.")
	}

	for i, key := range cacheKeys1 {
		if key != cacheKeys2[i] {
			errors.GenerateError(t, fmt.Sprintf("The cache key %d should match %s, %s given.", i, key, cacheKeys2[i]))
		}
	}
	for i, key := range surrogateKeys1 {
		if key != surrogateKeys2[i] {
			errors.GenerateError(t, fmt.Sprintf("The surrogate key %d should match %s, %s given.", i, key, surrogateKeys2[i]))
		}
	}
}
