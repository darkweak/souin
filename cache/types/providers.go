package types

import (
	"github.com/darkweak/souin/configurationtypes"
	"time"
)

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	ListKeys() []string
	Get(key string) []byte
	Set(key string, value []byte, url configurationtypes.URL, duration time.Duration)
	Delete(key string)
	Init() error
}
