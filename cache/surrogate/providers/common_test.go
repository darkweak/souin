package providers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"go.uber.org/zap"
)

const (
	baseHeaderValue  = "test0, test1,   test2,  test3, test4"
	emptyHeaderValue = ""
)

func mockCommonProvider() *baseStorage {
	sss := &SouinSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    &sync.Map{},
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         &sync.Mutex{},
			logger:     zap.NewNop(),
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

	bs.Storage.Store("test0", "first,second")
	bs.Storage.Store("STALE_test0", "STALTE_first,STALE_second")
	bs.Storage.Store("test2", "third,fourth")
	bs.Storage.Store("test5", "first,second,fifth")
	bs.Storage.Store("testInvalid", "invalid")
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
	bs.Storage.Store("test0", "first,second")
	bs.Storage.Store("test2", "third,fourth")
	bs.Storage.Store("test5", "first,second,fifth")
	bs.Storage.Store("testInvalid", "invalid")

	if e = bs.Store(&res, "stored"); e != nil {
		errors.GenerateError(t, "It shouldn't throw an error with a valid key.")
	}

	for i := 0; i < 5; i++ {
		value, _ := bs.Storage.Load(fmt.Sprintf("test%d", i))
		if !strings.Contains(value.(string), "stored") {
			errors.GenerateError(t, fmt.Sprintf("The key test%d must include stored, %s given.", i, value.(string)))
		}
	}

	value, _ := bs.Storage.Load("testInvalid")
	if strings.Contains(value.(string), "stored") {
		errors.GenerateError(t, "The surrogate storage should not contain stored.")
	}

	bs = mockCommonProvider()
	res.Header.Set(surrogateKey, "something")
	_ = bs.Store(&res, "/something")
	_ = bs.Store(&res, "/something")
	res.Header.Set(surrogateKey, "something")
	_ = bs.Store(&res, "/some")

	storageSize := 0
	bs.Storage.Range(func(_, _ any) bool {
		storageSize++
		return true
	})

	if storageSize != 2 {
		errors.GenerateError(t, "The surrogate storage should contain 2 stored elements.")
	}

	value, _ = bs.Storage.Load("STALE_something")
	if value.(string) != ",STALE_%2Fsomething,STALE_%2Fsome" {
		errors.GenerateError(t, "The STALE_something surrogate storage entry must contain 2 elements ,STALE_%2Fsomething,STALE_%2Fsome.")
	}
	value, _ = bs.Storage.Load("something")
	if value.(string) != ",%2Fsomething,%2Fsome" {
		errors.GenerateError(t, "The something surrogate storage entry must contain 2 elements ,%2Fsomething,%2Fsome.")
	}
}

func TestBaseStorage_Store_Load(t *testing.T) {
	var wg sync.WaitGroup
	res := http.Response{
		Header: http.Header{},
	}
	bs := mockCommonProvider()

	length := 4000
	for i := 0; i < length; i++ {
		wg.Add(1)
		go func(r http.Response, iteration int, group *sync.WaitGroup) {
			defer wg.Done()
			bs.Store(&r, fmt.Sprintf("my_dynamic_cache_key_%d", iteration))
		}(res, i, &wg)
	}

	wg.Wait()
	v, _ := bs.Storage.Load("")

	if len(strings.Split(v.(string), ",")) != length+1 {
		errors.GenerateError(t, fmt.Sprintf("The surrogate storage should contain %d stored elements, %d given.", length+1, len(strings.Split(v.(string), ","))))
	}
}
