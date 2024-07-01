package types

import (
	"net/http"
	"time"

	"github.com/darkweak/storages/core"
)

type Storer interface {
	MapKeys(prefix string) map[string]string
	ListKeys() []string
	Get(key string) []byte
	Set(key string, value []byte, duration time.Duration) error
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Name() string
	Reset() error

	// Multi level storer to handle fresh/stale at once
	GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response)
	SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error
}
