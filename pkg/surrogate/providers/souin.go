package providers

import (
	"github.com/darkweak/souin/configurationtypes"
)

// SouinSurrogateStorage is the layer for Surrogate-key support storage
type SouinSurrogateStorage struct {
	*baseStorage
}

func generateSouinInstance(config configurationtypes.AbstractConfigurationInterface, defaultStorerName string) *SouinSurrogateStorage {
	s := &SouinSurrogateStorage{baseStorage: &baseStorage{}}

	s.init(config, defaultStorerName)
	s.parent = s

	return s
}

func (*SouinSurrogateStorage) getHeaderSeparator() string {
	return ", "
}
