package providers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/tests"
	"github.com/darkweak/storages/core"
	"go.uber.org/zap"
)

const (
	baseHeaderValue  = "test0, test1,   test2,  test3, test4"
	emptyHeaderValue = ""
)

func mockCommonProvider() *baseStorage {
	memoryStorer, _ := storage.Factory(mockConfiguration(tests.BaseConfiguration))
	core.RegisterStorage(memoryStorer)
	config := tests.MockConfiguration(tests.NutsConfiguration)
	config.DefaultCache.Badger.Configuration = nil
	sss := &SouinSurrogateStorage{
		baseStorage: &baseStorage{
			Storage:    memoryStorer,
			Keys:       make(map[string]configurationtypes.SurrogateKeys),
			keysRegexp: make(map[string]keysRegexpInner),
			dynamic:    true,
			mu:         sync.Mutex{},
			logger:     zap.NewNop().Sugar(),
		},
	}

	sss.parent = sss

	return sss.baseStorage
}

func TestBaseStorage_ParseHeaders(t *testing.T) {
	bs := mockCommonProvider()

	fields := bs.ParseHeaders(baseHeaderValue)

	if len(fields) != 5 {
		t.Errorf("The fields length should be 5, %d given.", len(fields))
	}

	for i := 0; i < len(fields); i++ {
		expected := fmt.Sprintf("test%d", i)
		if fields[i] != expected {
			t.Errorf("The field number %d should be equal to %s, %s given.", i, expected, fields[i])
		}
	}
}

func TestBaseStorage_Purge(t *testing.T) {
	bs := mockCommonProvider()
	headerMock := http.Header{}
	headerMock.Set(surrogateKey, baseHeaderValue)

	tags, surrogates := bs.Purge(headerMock)
	if len(tags) != 0 {
		t.Error("The tags length should be empty.")
	}
	if len(surrogates) != 5 {
		t.Error("The surrogates length should be equal to 5.")
	}

	headerMock.Set(surrogateKey, emptyHeaderValue)

	tags, surrogates = bs.Purge(headerMock)
	if len(tags) != 0 {
		t.Error("The tags length should be empty.")
	}
	if len(surrogates) != 1 {
		t.Error("The surrogates length should be equal to 0.")
	}

	_ = bs.Storage.Set(surrogatePrefix+"test0", []byte("first,second"), storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set(surrogatePrefix+"STALE_test0", []byte("STALE_first,STALE_second"), storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set(surrogatePrefix+"test2", []byte("third,fourth"), storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set(surrogatePrefix+"test5", []byte("first,second,fifth"), storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set(surrogatePrefix+"testInvalid", []byte("invalid"), storageToInfiniteTTLMap[bs.Storage.Name()])

	headerMock.Set(surrogateKey, baseHeaderValue)
	tags, surrogates = bs.Purge(headerMock)

	if len(tags) != 4 {
		t.Error("The tags length should be equal to 4.")
	}
	if len(surrogates) != 5 {
		t.Error("The surrogates length should be equal to 5.")
	}
}

func TestBaseStorage_Store(t *testing.T) {
	res := http.Response{
		Header: http.Header{},
	}

	res.Header.Set(surrogateKey, baseHeaderValue)

	bs := mockCommonProvider()

	e := bs.Store(&res, "((((invalid_key_but_escaped", "")
	if e != nil {
		t.Error("It shouldn't throw an error with a valid key.")
	}

	_ = bs.Storage.Set("test0", []byte("first,second"), storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("test2", []byte("third,fourth"), storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("test5", []byte("first,second,fifth"), storageToInfiniteTTLMap[bs.Storage.Name()])
	_ = bs.Storage.Set("testInvalid", []byte("invalid"), storageToInfiniteTTLMap[bs.Storage.Name()])

	if e = bs.Store(&res, "stored", ""); e != nil {
		t.Error("It shouldn't throw an error with a valid key.")
	}

	for i := 0; i < 5; i++ {
		_ = bs.Storage.Get(fmt.Sprintf(surrogatePrefix+"test%d", i))
		// if !strings.Contains(string(value), "stored") {
		// 	// t.Errorf("The key %stest%d must include stored, %s given.", surrogatePrefix, i, string(value))
		// }
	}

	value := bs.Storage.Get("testInvalid")
	if strings.Contains(string(value), "stored") {
		t.Error("The surrogate storage should not contain stored.")
	}

	res.Header.Set(surrogateKey, "something")
	_ = bs.Store(&res, "/something", "")
	_ = bs.Store(&res, "/something", "")
	res.Header.Set(surrogateKey, "something")
	_ = bs.Store(&res, "/some", "")

	_ = len(bs.Storage.MapKeys(surrogatePrefix))
	// if storageSize != 6 {
	// 	// t.Errorf("The surrogate storage should contain 6 stored elements, %v given: %#v.\n", storageSize, bs.Storage.MapKeys(""))
	// }

	// value = bs.Storage.Get(surrogatePrefix + "something")
	// if string(value) != ",%2Fsomething,%2Fsome" {
	// 	t.Errorf("The something surrogate storage entry must contain 2 elements %s.", ",%2Fsomething,%2Fsome")
	// }
}

func TestBaseStorage_Store_Load(t *testing.T) {
	var wg sync.WaitGroup
	res := http.Response{
		Header: http.Header{},
	}
	bs := mockCommonProvider()

	length := 3000
	for i := 0; i < length; i++ {
		wg.Add(1)
		go func(r http.Response, iteration int, group *sync.WaitGroup) {
			defer wg.Done()
			_ = bs.Store(&r, fmt.Sprintf("my_dynamic_cache_key_%d", iteration), "")
		}(res, i, &wg)
	}

	wg.Wait()
	_ = bs.Storage.Get(surrogatePrefix)

	// if len(strings.Split(string(v), ",")) != length+1 {
	// 	// t.Errorf("The surrogate storage should contain %d stored elements, %d given.", length+1, len(strings.Split(string(v), ",")))
	// }
}

func TestContainsCacheKey(t *testing.T) {
	testCases := []struct {
		name         string
		currentValue string
		cacheKey     string
		expected     bool
	}{
		{"empty current value", "", "key1", false},
		{"exact match single key", "key1", "key1", true},
		{"key at beginning", "key1,key2,key3", "key1", true},
		{"key at end", "key1,key2,key3", "key3", true},
		{"key in middle", "key1,key2,key3", "key2", true},
		{"key not present", "key1,key2,key3", "key4", false},
		{"partial match should not match", "key1,key2,key3", "key", false},
		{"partial match at start should not match", "key12,key2,key3", "key1", false},
		{"partial match at end should not match", "key1,key2,key34", "key3", false},
		{"url encoded key present", "%2Fapi%2Fusers,other", "%2Fapi%2Fusers", true},
		{"url encoded key not present", "%2Fapi%2Fusers,other", "%2Fapi%2Fposts", false},
		{"key with special chars", "a%2Fb,c%2Fd", "a%2Fb", true},
		{"empty key in empty value", "", "", false},
		{"empty key in non-empty value", "key1,key2", "", false},
		{"similar keys should not match", "product-123,product-1234", "product-12", false},
		{"leading comma in stored value", ",key1,key2", "key1", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := containsCacheKey(tc.currentValue, tc.cacheKey)
			if result != tc.expected {
				t.Errorf("containsCacheKey(%q, %q) = %v, expected %v",
					tc.currentValue, tc.cacheKey, result, tc.expected)
			}
		})
	}
}
