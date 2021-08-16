package types

import (
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"time"
)

// AbstractProviderInterface should be implemented in any providers
type AbstractProviderInterface interface {
	ListKeys() []string
	Prefix(key string, req *http.Request) []byte
	Get(key string) []byte
	Set(key string, value []byte, url configurationtypes.URL, duration time.Duration)
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Reset()
}
