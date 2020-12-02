package types

import "github.com/darkweak/souin/configurationtypes"

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	GetRequestInCache(key string) ReverseResponse
	SetRequestInCache(key string, value []byte, url configurationtypes.URL)
	DeleteRequestInCache(key string)
	Init() error
}
