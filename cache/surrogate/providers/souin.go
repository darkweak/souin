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
	var storage map[string]string

	s := &SouinSurrogateStorage{}

	if len(config.GetSurrogateKeys()) == 0 {
		s.Keys = config.GetSurrogateKeys()
	}

	s.Storage = storage
	s.parent = s

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
