package types

import (
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
)

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	ListKeys() []string
	Prefix(key string, req *http.Request) []byte
	Get(key string) []byte
	Set(key string, value []byte, url configurationtypes.URL, duration time.Duration) error
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Reset() error
}

type AbstractReconnectProvider interface {
	AbstractProviderInterface
	Reconnect()
}
