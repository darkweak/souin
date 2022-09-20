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

func mockAkamaiProvider() *AkamaiSurrogateStorage {
	ass := &AkamaiSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    &sync.Map{},
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         &sync.Mutex{},
			logger:     zap.NewNop(),
		},
		url: "http://akamai/invalidate_tag",
	}

	ass.baseStorage.parent = ass

	return ass
}

func TestAkamaiSurrogateStorage_Purge(t *testing.T) {
	ap := mockAkamaiProvider()
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

func TestAkamaiSurrogateStorage_Store(t *testing.T) {
	ap := mockAkamaiProvider()
	res := http.Response{
		Header: http.Header{},
	}
	res.Header.Set(surrogateKey, baseHeaderValue)

	var e error
	if e = ap.Store(&res, "stored"); e != nil {
		errors.GenerateError(t, "It should not throw an error while store.")
	}

	edgeTagValue := res.Header.Get(edgeCacheTag)
	if edgeTagValue != baseHeaderValue {
		errors.GenerateError(t, fmt.Sprintf("EdgeTag should match %s, %s given.", baseHeaderValue, edgeTagValue))
	}

	if res.Header.Get(surrogateKey) != "" {
		errors.GenerateError(t, "The response should not contains the Surrogate keys header.")
	}

	if res.Header.Get(surrogateControl) != "" {
		errors.GenerateError(t, "The response should not contains the Surrogate control header.")
	}
}
