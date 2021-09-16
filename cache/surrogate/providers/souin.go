package providers

import (
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"strings"
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

func (_ *SouinSurrogateStorage) getOrderedSurrogateKeyHeadersCandidate() []string {
	return []string{
		surrogateKey,
	}
}

func (_ *SouinSurrogateStorage) getOrderedSurrogateControlHeadersCandidate() []string {
	return []string{
		souinCacheControl,
		surrogateControl,
		cdnCacheControl,
		cacheControl,
	}
}

func (s *SouinSurrogateStorage) getSurrogateControl(header http.Header) string {
	return getCandidateHeader(header, s.getOrderedSurrogateControlHeadersCandidate)
}

func (s *SouinSurrogateStorage) getSurrogateKey(header http.Header) string {
	return getCandidateHeader(header, s.getOrderedSurrogateKeyHeadersCandidate)
}

func (_ *SouinSurrogateStorage) candidateStore(tag string) bool {
	return !strings.Contains(tag, noStoreDirective)
}

func (s *SouinSurrogateStorage) Store(header *http.Header, cacheKey string) error {
	e := s.baseStorage.Store(header, cacheKey)
	header.Del(surrogateKey)
	header.Del(surrogateControl)

	return e
}
