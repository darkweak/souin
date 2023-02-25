package providers

import (
	"net/http"
	"net/url"

	"github.com/darkweak/souin/configurationtypes"
)

// FastlySurrogateStorage is the layer for Surrogate-key support storage
type FastlySurrogateStorage struct {
	*baseStorage
	providerAPIKey string
	serviceID      string
	strategy       string
}

func generateFastlyInstance(config configurationtypes.AbstractConfigurationInterface) *FastlySurrogateStorage {
	cdn := config.GetDefaultCache().GetCDN()
	f := &FastlySurrogateStorage{
		baseStorage:    &baseStorage{},
		providerAPIKey: cdn.APIKey,
		serviceID:      cdn.ServiceID,
		strategy:       "0",
	}

	if cdn.Strategy == "soft" {
		f.strategy = "1"
	}

	f.init(config)
	f.parent = f

	return f
}

func (*FastlySurrogateStorage) getHeaderSeparator() string {
	return " "
}

func (*FastlySurrogateStorage) getOrderedSurrogateKeyHeadersCandidate() []string {
	return []string{
		surrogateKey,
	}
}

func (*FastlySurrogateStorage) getOrderedSurrogateControlHeadersCandidate() []string {
	return []string{
		fastlyCacheControl,
		souinCacheControl,
		surrogateControl,
		cdnCacheControl,
		cacheControl,
	}
}

// Purge purges the urls associated to the tags
func (f *FastlySurrogateStorage) Purge(header http.Header) (cacheKeys []string, surrogateKeys []string) {
	keys, headers := f.baseStorage.Purge(header)
	req, err := http.NewRequest(http.MethodPost, "https://api.fastly.com/service/"+f.serviceID+"/purge", nil)
	if err == nil {
		req.Header.Set("Fastly-Soft-Purge", f.strategy)
		req.Header.Set("Fastly-Key", f.providerAPIKey)
		req.Header.Set("Accept", "application/json")

		go func() {
			for _, h := range headers {
				computedURL := "/service/" + f.serviceID + "/purge/" + h
				req.RequestURI = computedURL
				if req.URL, err = url.Parse(computedURL); err == nil {
					_, _ = new(http.Client).Do(req)
				}
			}
		}()
	}

	return keys, headers
}
