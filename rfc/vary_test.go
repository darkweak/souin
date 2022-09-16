package rfc

import (
	"bytes"
	ctx "context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/context"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

func TestVaryMatches(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	tr := NewTransport(prs, ykeys.InitializeYKeys(c.Ykeys), surrogate.InitializeSurrogate(c))

	r := httptest.NewRequest("GET", "http://domain.com/testing", nil)
	res := httptest.NewRecorder().Result()
	co := context.GetContext()
	co.Init(c)
	r = co.SetContext(r)

	if !varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return true if no header sent")
	}
	if !validateVary(r, res, r.Context().Value(context.Key).(string), tr) {
		errors.GenerateError(t, fmt.Sprintf("It doesn't contain vary header in the Response. It should validate it, %v given", res.Header))
	}

	header := "Cache"
	r.Header.Set(header, "same")
	res.Header.Set("vary", header)

	if !varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return true if Response contains a vary header that is not null in the request")
	}

	if !validateVary(r, res, r.Context().Value(context.Key).(string), tr) {
		errors.GenerateError(t, fmt.Sprintf("It contains valid vary headers in the Response. It should validate it, %v given", res.Header))
	}

	r.Header.Set(header, "")

	if varyMatches(res, r) {
		errors.GenerateError(t, "Vary match should return false if Response contains a vary header that is empty in the request")
	}

	if !validateVary(r, res, r.Context().Value(context.Key).(string), tr) {
		errors.GenerateError(t, fmt.Sprintf("It contains valid vary headers in the Response. It should validate it, %v given", res.Header))
	}

	if len(prs.Get(r.Context().Value(context.Key).(string))) == 0 {
		errors.GenerateError(t, fmt.Sprintf("The key %s should exist in the storage provider. %v given", r.Context().Value(context.Key).(string), prs.Get(r.Context().Value(context.Key).(string))))
	}

	variedHeaders := headerAllCommaSepValues(res.Header)
	variedCacheKey := GetVariedCacheKey(r, variedHeaders)
	b := prs.Get(GetVariedCacheKey(r, headerAllCommaSepValues(res.Header)))
	if len(b) == 0 {
		errors.GenerateError(t, fmt.Sprintf("The key %s with headers %v should exist in the storage provider. %v given", variedCacheKey, variedHeaders, b))
	}
}

func TestValidateVary_Load(t *testing.T) {
	if validateVary(nil, nil, "", nil) {
		errors.GenerateError(t, "The validateVary must return false when a nil response is passed as parameter.")
	}

	response := &http.Response{
		Header: http.Header{},
	}
	response.Header.Set("Vary", "X-Varied")
	response.Header.Set("X-Varied", "value")
	response.Header.Set("Surrogate-Key", "souin_test")
	response.Body = io.NopCloser(bytes.NewBuffer([]byte("Hello world!")))

	badger, _ := providers.BadgerConnectionFactory(tests.MockConfiguration(tests.CDNConfiguration))
	transport := NewTransport(
		badger,
		ykeys.InitializeYKeys(make(map[string]configurationtypes.SurrogateKeys)),
		surrogate.InitializeSurrogate(tests.MockConfiguration(tests.CDNConfiguration)),
	)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(ctx.WithValue(req.Context(), context.CacheName, "Souin"))
	req = req.WithContext(ctx.WithValue(req.Context(), context.Key, "GET-domain.com-/something"))
	if !validateVary(req, response, "", transport) {
		errors.GenerateError(t, "The validateVary must return true when a valid response is passed as parameter.")
	}

	if response.Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored" {
		errors.GenerateError(t, "The validateVary must set the stored Cache-Status header.")
	}

	if len(strings.Split(transport.SurrogateStorage.List()["souin_test"], ",")) != 2 || len(strings.Split(transport.SurrogateStorage.List()["STALE_souin_test"], ",")) != 2 {
		errors.GenerateError(t, "The surrogate storage must contain 2 items in souin_test and STALE_souin_test")
	}
	var wg sync.WaitGroup

	length := 4000
	for i := 0; i < length; i++ {
		wg.Add(1)
		go func(iteration int, group *sync.WaitGroup) {
			defer wg.Done()

			rs := &http.Response{
				Header: http.Header{},
			}
			rs.Header.Set("Surrogate-Key", "souin_test")

			rq := req.WithContext(ctx.WithValue(req.Context(), context.CacheName, "Souin"))
			if !validateVary(rq, rs, fmt.Sprintf("sk_%d", iteration), transport) {
				errors.GenerateError(t, "The validateVary must return true when a valid response is passed as parameter.")
			}
		}(i, &wg)
	}

	wg.Wait()

	if len(strings.Split(transport.SurrogateStorage.List()["souin_test"], ",")) != 4002 || len(strings.Split(transport.SurrogateStorage.List()["STALE_souin_test"], ",")) != 4002 {
		errors.GenerateError(t, "The surrogate storage must contain 4002 items in souin_test and STALE_souin_test")
	}

	flushRq, _ := http.NewRequest("PURGE", "http://domain.com/souin-api/souin", nil)
	souinAPI := api.Initialize(transport, tests.MockConfiguration(tests.CDNConfiguration))

	flushRq.Header.Set("Surrogate-Key", "souin_test")
	souinAPI[1].HandleRequest(httptest.NewRecorder(), flushRq)

	if len(transport.SurrogateStorage.List()) != 0 {
		errors.GenerateError(t, `The surrogate list must be empty because it had "" and "STALE_" that should be deleted.`)
	}
	if len(transport.GetProvider().ListKeys()) != 6 {
		errors.GenerateError(t, `The provider must have all related keys deleted.`)
	}
}
