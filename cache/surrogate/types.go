package surrogate

import (
	"github.com/darkweak/souin/configurationtypes"
	"github.com/dgraph-io/ristretto"
	"net/http"
)

const (
	SurrogateKeys = "Surrogate-Keys"
	SurrogateControl = "Surrogate-Control"
)

// SurrogateStorage is the layer for Surrogate-key support storage
type SurrogateStorage struct {
	*ristretto.Cache
	Keys    map[string]configurationtypes.YKey
	dynamic bool
}

func (s *SurrogateStorage) Store(header http.Header) {
	header.Get(SurrogateKeys)
}

func (s *SurrogateStorage) purgeTag(tag string) {
	s.Del(tag)
}

func (s *SurrogateStorage) Purge(header http.Header) {
	header.Values(SurrogateControl)
}
