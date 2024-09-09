package providers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/darkweak/souin/configurationtypes"
)

// CloudflareSurrogateStorage is the layer for Surrogate-key support storage
type CloudflareSurrogateStorage struct {
	*baseStorage
	providerAPIKey string
	email          string
	zoneID         string
}

func generateCloudflareInstance(config configurationtypes.AbstractConfigurationInterface, defaultStorerName string) *CloudflareSurrogateStorage {
	cdn := config.GetDefaultCache().GetCDN()
	f := &CloudflareSurrogateStorage{
		baseStorage:    &baseStorage{},
		providerAPIKey: cdn.APIKey,
		zoneID:         cdn.ZoneID,
		email:          cdn.Email,
	}

	f.init(config, defaultStorerName)
	f.parent = f

	return f
}

func (*CloudflareSurrogateStorage) getHeaderSeparator() string {
	return ","
}

// Store stores the response tags located in the first non empty supported header
func (c *CloudflareSurrogateStorage) Store(response *http.Response, cacheKey, uri string) error {
	defer func() {
		response.Header.Del(surrogateKey)
		response.Header.Del(surrogateControl)
	}()
	e := c.baseStorage.Store(response, cacheKey, uri)
	response.Header.Set(cacheTag, strings.Join(c.ParseHeaders(response.Header.Get(surrogateKey)), c.getHeaderSeparator()))

	return e
}

func (*CloudflareSurrogateStorage) getOrderedSurrogateKeyHeadersCandidate() []string {
	return []string{
		cacheTag,
		surrogateKey,
	}
}

func processBatches(arr []string, req *http.Request) {
	const maxPerBatch = 30
	for i := 0; i < len(arr); i += maxPerBatch {
		j := i + maxPerBatch
		if j > len(arr) {
			j = len(arr)
		}

		body, _ := json.Marshal(map[string]interface{}{"tags": arr[i:j]})
		req.Body = io.NopCloser(bytes.NewBuffer(body))
		_, _ = new(http.Client).Do(req)
	}
}

// Purge purges the urls associated to the tags
func (c *CloudflareSurrogateStorage) Purge(header http.Header) (cacheKeys []string, surrogateKeys []string) {
	keys, headers := c.baseStorage.Purge(header)
	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.cloudflare.com/client/v4/zones/"+c.zoneID+"/purge",
		nil,
	)
	if err == nil {
		req.Header.Set("X-Auth-Email", c.email)
		req.Header.Set("X-Auth-Key", c.providerAPIKey)
		req.Header.Set("Content-Type", "application/json")

		go func() {
			processBatches(headers, req)
		}()
	}

	return keys, headers
}
