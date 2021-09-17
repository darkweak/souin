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
	var storage map[string]string

	cdn := config.GetDefaultCache().GetCDN()
	f := &AkamaiSurrogateStorage{
		baseStorage: &baseStorage{},
	}

	strategy := "delete"
	if cdn.Strategy == "soft" {
		strategy = "invalidate"
	}

	f.url = "https://" + cdn.Hostname + "/ccu/v3/" + strategy + "/tag"
	if cdn.Network != "" {
		f.url += "/" + cdn.Network
	}

	if len(config.GetSurrogateKeys()) != 0 {
		f.Keys = config.GetSurrogateKeys()
	}

	f.Storage = storage
	f.parent = f

	return f
}

func (*AkamaiSurrogateStorage) getHeaderSeparator() string {
	return ", "
}

// Store stores the response tags located in the first non empty supported header
func (f *AkamaiSurrogateStorage) Store(header *http.Header, cacheKey string) error {
	e := f.baseStorage.Store(header, cacheKey)
	header.Set(edgeCacheTag, header.Get(surrogateKey))
	header.Del(surrogateKey)
	header.Del(surrogateControl)

	return e
}

// Purge purges the urls associated to the tags
func (f *AkamaiSurrogateStorage) Purge(header http.Header) (cacheKeys []string, surrogateKeys []string) {
	keys, headers := f.baseStorage.Purge(header)
	m, b := map[string][]string{"objects": headers}, new(bytes.Buffer)
	e := json.NewEncoder(b).Encode(m)
	if e != nil {
		return keys, headers
	}
	req, err := http.NewRequest(http.MethodPost, f.url, b)
	if err == nil {
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")

		go func() {
			_, _ = new(http.Client).Do(req)
		}()
	}

	return keys, headers
}
