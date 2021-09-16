package providers

import (
	"net/http"
	"net/url"

	"github.com/darkweak/souin/configurationtypes"
)

// AkamaiSurrogateStorage is the layer for Surrogate-key support storage
type AkamaiSurrogateStorage struct {
	*baseStorage
	Keys           map[string]configurationtypes.SurrogateKeys
	providerApiKey string
	serviceId      string
	strategy       string
}

func generateAkamaiInstance(config configurationtypes.AbstractConfigurationInterface) *AkamaiSurrogateStorage {
	var storage map[string]string

	if len(config.GetSurrogateKeys()) == 0 {
		return nil
	}

	cdn := config.GetDefaultCache().GetCDN()
	f := &AkamaiSurrogateStorage{
		Keys:           config.GetSurrogateKeys(),
		providerApiKey: cdn.ApiKey,
		strategy:       "0",
	}

	if cdn.Strategy == "soft" {
		f.strategy = "1"
	}

	f.Storage = storage
	f.parent = f

	return f
}

func (_ *AkamaiSurrogateStorage) getHeaderSeparator() string {
	return ", "
}

func (f *AkamaiSurrogateStorage) Store(header *http.Header, cacheKey string) error {
	e := f.baseStorage.Store(header, cacheKey)
	header.Set(edgeCacheTag, header.Get(surrogateKey))
	header.Del(surrogateKey)
	header.Del(surrogateControl)

	return e
}

func (f *AkamaiSurrogateStorage) Purge(header http.Header) []string {
	headers := f.baseStorage.Purge(header)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "https://api.akamai.com/service/"+f.serviceId+"/purge", nil)
	if err == nil {
		req.Header.Set("Akamai-Soft-Purge", f.strategy)
		req.Header.Set("Akamai-Key", f.providerApiKey)
		req.Header.Set("Accept", "application/json")

		go func() {
			for _, h := range headers {
				computedURL := "/service/" + f.serviceId + "/purge/" + h
				req.RequestURI = computedURL
				if req.URL, err = url.Parse(computedURL); err == nil {
					_, _ = client.Do(req)
				}
			}
		}()
	}

	return headers
}
