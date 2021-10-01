package providers

import (
	"net/http"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

func mockSouinProvider() *SouinSurrogateStorage {
	sss := &SouinSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    make(map[string]string),
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
		},
	}

	sss.baseStorage.parent = sss

	return sss
}

func TestSouinSurrogateStorage_Store(t *testing.T) {
	sp := mockSouinProvider()
	req := http.Request{
		Header: http.Header{},
	}
	req.Header.Set(surrogateKey, baseHeaderValue)

	var e error
	if e = sp.Store(&req, "stored"); e != nil {
		errors.GenerateError(t, "It should not throw an error while store.")
	}

	if req.Header.Get(surrogateKey) != "" {
		errors.GenerateError(t, "The request should not contains the Surrogate keys header.")
	}

	if req.Header.Get(surrogateControl) != "" {
		errors.GenerateError(t, "The request should not contains the Surrogate control header.")
	}
}
