package providers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
)

const (
	baseHeaderValue  = "test0, test1,   test2,  test3, test4"
	emptyHeaderValue = ""
)

func mockCommonProvider() *baseStorage {
	sss := &SouinSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    make(map[string]string),
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         &sync.Mutex{},
		},
	}

	sss.baseStorage.parent = sss

	return sss.baseStorage
}

func TestBaseStorage_ParseHeaders(t *testing.T) {
	bs := mockCommonProvider()

	fields := bs.ParseHeaders(baseHeaderValue)

	if len(fields) != 5 {
		errors.GenerateError(t, fmt.Sprintf("The fields length should be 5, %d given.", len(fields)))
	}

	for i := 0; i < len(fields); i++ {
		expected := fmt.Sprintf("test%d", i)
		if fields[i] != expected {
			errors.GenerateError(t, fmt.Sprintf("The field number %d should be equal to %s, %s given.", i, expected, fields[i]))
		}
	}
}

func TestBaseStorage_Purge(t *testing.T) {
	bs := mockCommonProvider()
	headerMock := http.Header{}
	headerMock.Set(surrogateKey, baseHeaderValue)

	tags, surrogates := bs.Purge(headerMock)
	if len(tags) != 0 {
		errors.GenerateError(t, "The tags length should be empty.")
	}
	if len(surrogates) != 5 {
		errors.GenerateError(t, "The surrogates length should be equal to 0.")
	}

	headerMock.Set(surrogateKey, emptyHeaderValue)

	tags, surrogates = bs.Purge(headerMock)
	if len(tags) != 0 {
		errors.GenerateError(t, "The tags length should be empty.")
	}
	if len(surrogates) != 1 {
		errors.GenerateError(t, "The surrogates length should be equal to 0.")
	}

	bs.Storage["test0"] = "first,second"
	bs.Storage["STALE_test0"] = "STALTE_first,STALE_second"
	bs.Storage["test2"] = "third,fourth"
	bs.Storage["test5"] = "first,second,fifth"
	bs.Storage["testInvalid"] = "invalid"
	headerMock.Set(surrogateKey, baseHeaderValue)
	tags, surrogates = bs.Purge(headerMock)

	if len(tags) != 6 {
		errors.GenerateError(t, "The tags length should be equal to 6.")
	}
	if len(surrogates) != 5 {
		errors.GenerateError(t, "The surrogates length should be equal to 5.")
	}
}

func TestBaseStorage_Store(t *testing.T) {
	res := http.Response{
		Header: http.Header{},
	}

	res.Header.Set(surrogateKey, baseHeaderValue)

	bs := mockCommonProvider()

	e := bs.Store(&res, "((((invalid_key_but_escaped")
	if e != nil {
		errors.GenerateError(t, "It shouldn't throw an error with a valid key.")
	}

	bs = mockCommonProvider()
	bs.Storage["test0"] = "first,second"
	bs.Storage["test2"] = "third,fourth"
	bs.Storage["test5"] = "first,second,fifth"
	bs.Storage["testInvalid"] = "invalid"

	if e = bs.Store(&res, "stored"); e != nil {
		errors.GenerateError(t, "It shouldn't throw an error with a valid key.")
	}

	for i := 0; i < 5; i++ {
		if !strings.Contains(bs.Storage[fmt.Sprintf("test%d", i)], "stored") {
			errors.GenerateError(t, fmt.Sprintf("The key test%d must include stored, %s given.", i, bs.Storage[fmt.Sprintf("test%d", i)]))
		}
	}

	if strings.Contains(bs.Storage["testInvalid"], "stored") {
		errors.GenerateError(t, "The surrogate storage should not contain stored.")
	}
}
