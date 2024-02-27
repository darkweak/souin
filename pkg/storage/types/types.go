package types

import (
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
)

type KeyIndex struct {
	StoredAt      time.Time   `json:"stored"`
	FreshTime     time.Time   `json:"fresh"`
	StaleTime     time.Time   `json:"stale"`
	VariedHeaders http.Header `json:"varied"`
	Etag          string      `json:"etag"`
}

type StorageMapper struct {
	Mapping map[string]KeyIndex `json:"mapping"`
}

type Storer interface {
	MapKeys(prefix string) map[string]string
	ListKeys() []string
	Prefix(key string, req *http.Request, validator *rfc.Revalidator) *http.Response
	Get(key string) []byte
	Set(key string, value []byte, url configurationtypes.URL, duration time.Duration) error
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Name() string
	Reset() error

	// Multi level storer to handle fresh/stale at once
	GetMultiLevel(key string, req *http.Request, validator *rfc.Revalidator) (fresh *http.Response, stale *http.Response)
	SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration) error
}
