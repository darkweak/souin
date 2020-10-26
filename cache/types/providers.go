package types

import configuration_types "github.com/darkweak/souin/configuration_types"

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	GetRequestInCache(key string) ReverseResponse
	SetRequestInCache(key string, value []byte, url configuration_types.URL)
	DeleteRequestInCache(key string)
	Init() error
}
