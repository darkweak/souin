package types

import "github.com/darkweak/souin/configuration"

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	GetRequestInCache(key string) ReverseResponse
	SetRequestInCache(key string, value []byte, url configuration.URL)
	DeleteRequestInCache(key string)
	Init() error
}
