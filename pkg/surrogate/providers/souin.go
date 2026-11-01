package providers

import (
	"github.com/darkweak/souin/configurationtypes"
	"time"
)

// SouinSurrogateStorage is the layer for Surrogate-key support storage
type SouinSurrogateStorage struct {
	*baseStorage
}

func generateSouinInstance(config configurationtypes.AbstractConfigurationInterface, defaultStorerName string, defaultTTL time.Duration) *SouinSurrogateStorage {
	s := &SouinSurrogateStorage{baseStorage: &baseStorage{}}

	s.init(config, defaultStorerName, defaultTTL)
	s.parent = s

	return s
}

func (*SouinSurrogateStorage) getHeaderSeparator() string {
	return ", "
}
