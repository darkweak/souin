package providers

import (
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"net/url"
)

// FastlySurrogateStorage is the layer for Surrogate-key support storage
type FastlySurrogateStorage struct {
	*baseStorage
	Keys           map[string]configurationtypes.SurrogateKeys
	providerApiKey string
	serviceId      string
	strategy       string
}

func generateFastlyInstance(config configurationtypes.AbstractConfigurationInterface) *FastlySurrogateStorage {
	var storage map[string]string

	if len(config.GetSurrogateKeys()) == 0 {
		return nil
	}

	cdn := config.GetDefaultCache().GetCDN()
	f := &FastlySurrogateStorage{
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

func (_ *FastlySurrogateStorage) getHeaderSeparator() string {
	return " "
}

func (_ *FastlySurrogateStorage) getOrderedSurrogateKeyHeadersCandidate() []string {
	return []string{
		surrogateKey,
	}
}

func (_ *FastlySurrogateStorage) getOrderedSurrogateControlHeadersCandidate() []string {
	return []string{
		fastlyCacheControl,
		souinCacheControl,
		surrogateControl,
		cdnCacheControl,
		cacheControl,
	}
}

func (f *FastlySurrogateStorage) Store(header *http.Header, cacheKey string) error {
	return f.baseStorage.Store(header, cacheKey)
}

func (f *FastlySurrogateStorage) Purge(header http.Header) []string {
	headers := f.baseStorage.Purge(header)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, "https://api.fastly.com/service/"+f.serviceId+"/purge", nil)
	if err == nil {
		req.Header.Set("Fastly-Soft-Purge", f.strategy)
		req.Header.Set("Fastly-Key", f.providerApiKey)
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
