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
	s := &SouinSurrogateStorage{}

	s.init(config)
	s.parent = s

	return s
}

func (*SouinSurrogateStorage) getHeaderSeparator() string {
	return ", "
}

// Store stores the response tags located in the first non empty supported header
func (s *SouinSurrogateStorage) Store(request *http.Request, cacheKey string) error {
	e := s.baseStorage.Store(request, cacheKey)
	request.Header.Del(surrogateKey)
	request.Header.Del(surrogateControl)

	return e
}
