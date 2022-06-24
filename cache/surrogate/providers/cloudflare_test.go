package providers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

func mockCloudflareProvider() *CloudflareSurrogateStorage {
	ass := &CloudflareSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    make(map[string]string),
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         &sync.Mutex{},
		},
		providerAPIKey: "my_api_key",
		zoneID:         "Zone_id",
		email:          "client@email.com",
	}

	ass.baseStorage.parent = ass

	return ass
}

func TestCloudflareSurrogateStorage_Purge(t *testing.T) {
	cp := mockCloudflareProvider()
	headerMock := http.Header{}
	headerMock.Set(surrogateKey, baseHeaderValue)

	cacheKeys1, surrogateKeys1 := cp.baseStorage.Purge(headerMock)
	cacheKeys2, surrogateKeys2 := cp.Purge(headerMock)

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

func TestCloudflareSurrogateStorage_Store(t *testing.T) {
	cp := mockCloudflareProvider()
	res := http.Response{
		Header: http.Header{},
	}
	res.Header.Set(surrogateKey, baseHeaderValue)

	var e error
	if e = cp.Store(&res, "stored"); e != nil {
		errors.GenerateError(t, "It should not throw an error while store.")
	}

	cacheTagValue := res.Header.Get(cacheTag)
	if cacheTagValue == baseHeaderValue {
		errors.GenerateError(t, fmt.Sprintf("Cache-Tag should not match %s, %s given.", baseHeaderValue, cacheTagValue))
	}
	if cacheTagValue != strings.Join(cp.ParseHeaders(baseHeaderValue), cp.getHeaderSeparator()) {
		errors.GenerateError(t, fmt.Sprintf("Cache-Tag should match %s, %s given.", baseHeaderValue, cacheTagValue))
	}

	if res.Header.Get(surrogateKey) != "" {
		errors.GenerateError(t, "The response should not contains the Surrogate keys header.")
	}

	if res.Header.Get(surrogateControl) != "" {
		errors.GenerateError(t, "The response should not contains the Surrogate control header.")
	}
}
