package providers

import (
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
)

// SouinSurrogateStorage is the layer for Surrogate-key support storage
type SouinSurrogateStorage struct {
	*baseStorage
	Keys map[string]configurationtypes.SurrogateKeys
}

func generateSouinInstance(config configurationtypes.AbstractConfigurationInterface) *SouinSurrogateStorage {
	var storage map[string]string

	if len(config.GetSurrograteKeys()) == 0 {
		return nil
	}

	s := &SouinSurrogateStorage{
		Keys: config.GetSurrograteKeys(),
	}

	s.parent = s
	s.Storage = storage

	return s
}

func (_ *SouinSurrogateStorage) getHeaderSeparator() string {
	return ", "
}

func (s *SouinSurrogateStorage) Store(header *http.Header, cacheKey string) error {
	e := s.baseStorage.Store(header, cacheKey)
	header.Del(surrogateKey)
	header.Del(surrogateControl)

	return e
}
