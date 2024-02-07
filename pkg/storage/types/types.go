package types

import (
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
)

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
}
