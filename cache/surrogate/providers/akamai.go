package providers

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

// AkamaiSurrogateStorage is the layer for Surrogate-key support storage
type AkamaiSurrogateStorage struct {
	*baseStorage
	url string
}

func generateAkamaiInstance(config configurationtypes.AbstractConfigurationInterface) *AkamaiSurrogateStorage {
	cdn := config.GetDefaultCache().GetCDN()
	a := &AkamaiSurrogateStorage{baseStorage: &baseStorage{}}

	strategy := "delete"
	if cdn.Strategy == "soft" {
		strategy = "invalidate"
	}

	a.url = "https://" + cdn.Hostname + "/ccu/v3/" + strategy + "/tag"
	if cdn.Network != "" {
		a.url += "/" + cdn.Network
	}

	a.init(config)
	a.parent = a

	return a
}

func (*AkamaiSurrogateStorage) getHeaderSeparator() string {
	return ", "
}

// Store stores the response tags located in the first non empty supported header
func (a *AkamaiSurrogateStorage) Store(request *http.Request, cacheKey string) error {
	e := a.baseStorage.Store(request, cacheKey)
	request.Header.Set(edgeCacheTag, request.Header.Get(surrogateKey))
	request.Header.Del(surrogateKey)
	request.Header.Del(surrogateControl)

	return e
}

// Purge purges the urls associated to the tags
func (a *AkamaiSurrogateStorage) Purge(header http.Header) (cacheKeys []string, surrogateKeys []string) {
	keys, headers := a.baseStorage.Purge(header)
	m, b := map[string][]string{"objects": headers}, new(bytes.Buffer)
	e := json.NewEncoder(b).Encode(m)
	if e != nil {
		return keys, headers
	}
	req, err := http.NewRequest(http.MethodPost, a.url, b)
	if err == nil {
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		go func() {
			_, _ = new(http.Client).Do(req)
		}()
	}

	return keys, headers
}
