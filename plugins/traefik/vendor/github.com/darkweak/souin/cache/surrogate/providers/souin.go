package providers

import (
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

// SouinSurrogateStorage is the layer for Surrogate-key support storage
type SouinSurrogateStorage struct {
	*baseStorage
}

func generateSouinInstance(config configurationtypes.AbstractConfigurationInterface) *SouinSurrogateStorage {
	s := &SouinSurrogateStorage{baseStorage: &baseStorage{}}

	s.init(config)
	s.parent = s

	return s
}

func (*SouinSurrogateStorage) getHeaderSeparator() string {
	return ", "
}

// Store stores the response tags located in the first non empty supported header
func (s *SouinSurrogateStorage) Store(response *http.Response, cacheKey string) error {
	defer func() {
		response.Header.Del(surrogateKey)
		response.Header.Del(surrogateControl)
	}()

	e := s.baseStorage.Store(response, cacheKey)

	return e
}
