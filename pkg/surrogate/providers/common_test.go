package providers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/tests"
	"go.uber.org/zap"
)

const (
	baseHeaderValue  = "test0, test1,   test2,  test3, test4"
	emptyHeaderValue = ""
)

func mockCommonProvider() (*baseStorage, func() error) {
	instanciator, _ := storage.NewStorageFromName("nuts")
	config := tests.MockConfiguration(tests.NutsConfiguration)
	config.DefaultCache.Badger.Configuration = nil
	storer, _ := instanciator(config)
	sss := &SouinSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    storer,
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         &sync.Mutex{},
			logger:     zap.NewNop(),
		},
	}

	sss.baseStorage.parent = sss

	return sss.baseStorage, storer.(*storage.Nuts).Close
}

func TestBaseStorage_ParseHeaders(t *testing.T) {
	bs, _ := mockCommonProvider()

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
	bs, _ := mockCommonProvider()
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

	_ = bs.Storage.Set("test0", []byte("first,second"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("STALE_test0", []byte("STALE_first,STALE_second"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("test2", []byte("third,fourth"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("test5", []byte("first,second,fifth"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("testInvalid", []byte("invalid"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
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

	bs, _ := mockCommonProvider()

	e := bs.Store(&res, "((((invalid_key_but_escaped")
	if e != nil {
		errors.GenerateError(t, "It shouldn't throw an error with a valid key.")
	}

	_ = bs.Storage.Set("test0", []byte("first,second"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("test2", []byte("third,fourth"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("test5", []byte("first,second,fifth"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("testInvalid", []byte("invalid"), configurationtypes.URL{}, storageToInfiniteTTLMap[bs.Storage.Name()])

	if e = bs.Store(&res, "stored"); e != nil {
		errors.GenerateError(t, "It shouldn't throw an error with a valid key.")
	}

	for i := 0; i < 5; i++ {
		value := bs.Storage.Get(fmt.Sprintf(surrogatePrefix+"test%d", i))
		if !strings.Contains(string(value), "stored") {
			errors.GenerateError(t, fmt.Sprintf("The key %stest%d must include stored, %s given.", surrogatePrefix, i, string(value)))
		}
	}

	value := bs.Storage.Get("testInvalid")
	if strings.Contains(string(value), "stored") {
		errors.GenerateError(t, "The surrogate storage should not contain stored.")
	}

	res.Header.Set(surrogateKey, "something")
	_ = bs.Store(&res, "/something")
	_ = bs.Store(&res, "/something")
	res.Header.Set(surrogateKey, "something")
	_ = bs.Store(&res, "/some")

	storageSize := len(bs.Storage.ListKeys())

	if storageSize != 9 {
		errors.GenerateError(t, fmt.Sprintf("The surrogate storage should contain 9 stored elements, %v given: %#v.\n", storageSize, bs.Storage.ListKeys()))
	}

	value = bs.Storage.Get("SURROGATE_STALE_something")
	if string(value) != ",STALE_%2Fsomething,STALE_%2Fsome" {
		errors.GenerateError(t, "The STALE_something surrogate storage entry must contain 2 elements ,STALE_%2Fsomething,STALE_%2Fsome.")
	}
	value = bs.Storage.Get("SURROGATE_something")
	if string(value) != ",%2Fsomething,%2Fsome" {
		errors.GenerateError(t, "The something surrogate storage entry must contain 2 elements ,%2Fsomething,%2Fsome.")
	}
}

func TestBaseStorage_Store_Load(t *testing.T) {
	var wg sync.WaitGroup
	res := http.Response{
		Header: http.Header{},
	}
	bs, _ := mockCommonProvider()

	length := 3000
	for i := 0; i < length; i++ {
		wg.Add(1)
		go func(r http.Response, iteration int, group *sync.WaitGroup) {
			defer wg.Done()
			_ = bs.Store(&r, fmt.Sprintf("my_dynamic_cache_key_%d", iteration))
		}(res, i, &wg)
	}

	wg.Wait()
	v := bs.Storage.Get(surrogatePrefix)

	if len(strings.Split(string(v), ",")) != length+1 {
		errors.GenerateError(t, fmt.Sprintf("The surrogate storage should contain %d stored elements, %d given.", length+1, len(strings.Split(string(v), ","))))
	}
}
