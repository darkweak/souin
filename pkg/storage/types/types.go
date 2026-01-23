package types

import (
	"net/http"
	"time"

	"github.com/darkweak/storages/core"
)

const DefaultStorageName = "DEFAULT"
const OneYearDuration = 365 * 24 * time.Hour

type Storer interface {
	MapKeys(prefix string) map[string]string
	ListKeys() []string
	Get(key string) []byte
	Set(key string, value []byte, duration time.Duration) error
	Delete(key string)
	DeleteMany(key string)
	Init() error
	Name() string
	Uuid() string
	Reset() error

	// Multi level storer to handle fresh/stale at once
	GetMultiLevel(key string, req *http.Request, validator *core.Revalidator) (fresh *http.Response, stale *http.Response)
	SetMultiLevel(baseKey, variedKey string, value []byte, variedHeaders http.Header, etag string, duration time.Duration, realKey string) error
}

// MappingEvictor is an optional interface that storers can implement to provide
// efficient mapping eviction without requiring SCAN operations.
// Storers that implement this interface (e.g., Redis with sorted sets) can handle
// expired mapping entry cleanup more efficiently than the default SCAN-based approach.
// See: https://github.com/darkweak/souin/issues/671
type MappingEvictor interface {
	// EvictExpiredMappingEntries removes expired entries from mapping keys.
	// Returns true if the storer handles eviction natively (no further action needed).
	// Returns false if the default SCAN-based eviction should be used.
	EvictExpiredMappingEntries() bool
}
