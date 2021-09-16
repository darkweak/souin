package providers

import (
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

// SouinSurrogateStorage is the layer for Surrogate-key support storage
type SouinSurrogateStorage struct {
	*baseStorage
	Keys map[string]configurationtypes.SurrogateKeys
}

func generateSouinInstance(config configurationtypes.AbstractConfigurationInterface) *SouinSurrogateStorage {
	var storage map[string]string

	if len(config.GetSurrogateKeys()) == 0 {
		return nil
	}

	s := &SouinSurrogateStorage{
		Keys: config.GetSurrogateKeys(),
	}

	s.parent = s
	s.Storage = storage

	return s
}

func (*SouinSurrogateStorage) getHeaderSeparator() string {
	return ", "
}

// Store stores the response tags located in the first non empty supported header
func (s *SouinSurrogateStorage) Store(header *http.Header, cacheKey string) error {
	e := s.baseStorage.Store(header, cacheKey)
	header.Del(surrogateKey)
	header.Del(surrogateControl)

	return e
}
