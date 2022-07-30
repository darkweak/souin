package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkweak/souin/cache/surrogate"
	surrogate_providers "github.com/darkweak/souin/cache/surrogate/providers"
	"github.com/darkweak/souin/configurationtypes"

	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
)

func mockSouinAPI() *SouinAPI {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	config.DefaultCache.CDN = configurationtypes.CDN{
		Dynamic: true,
	}
	prs := providers.InitializeProvider(config)
	security := auth.InitializeSecurity(config)
	return &SouinAPI{
		"/souinbasepath",
		true,
		prs,
		security,
		ykeys.InitializeYKeys(config.Ykeys),
		surrogate.InitializeSurrogate(config),
	}
}

func TestSouinAPI_BulkDelete(t *testing.T) {
	souinMock := mockSouinAPI()
	souinMock.provider.Set("firstKey", []byte("value"), tests.GetMatchedURL("firstKey"), 20*time.Second)
	souinMock.provider.Set("secondKey", []byte("value"), tests.GetMatchedURL("secondKey"), 20*time.Second)
	time.Sleep(3 * time.Second)
	if len(souinMock.GetAll()) != 4 {
		errors.GenerateError(t, "Souin API should have a record")
	}
	souinMock.BulkDelete(".+")
	if len(souinMock.GetAll()) != 0 {
		errors.GenerateError(t, "Souin API shouldn't have a record")
	}
}

func TestSouinAPI_Delete(t *testing.T) {
	souinMock := mockSouinAPI()
	souinMock.provider.Set("key", []byte("value"), tests.GetMatchedURL("key"), 20*time.Second)
	time.Sleep(3 * time.Second)
	if len(souinMock.GetAll()) != 2 {
		errors.GenerateError(t, "Souin API should have 2 records")
	}
	souinMock.Delete("key")
	time.Sleep(1 * time.Second)
	if len(souinMock.GetAll()) > 1 {
		errors.GenerateError(t, "Souin API should contains only one record")
	}
	souinMock.Delete("STALE_key")
	time.Sleep(1 * time.Second)
	if len(souinMock.GetAll()) == 1 {
		errors.GenerateError(t, "Souin API shouldn't have a record")
	}
}

func TestSouinAPI_GetAll(t *testing.T) {
	souinMock := mockSouinAPI()
	if len(souinMock.GetAll()) > 0 {
		errors.GenerateError(t, "Souin API don't have any record yet")
	}

	souinMock.provider.Set("key", []byte("value"), tests.GetMatchedURL("key"), 6*time.Second)
	time.Sleep(3 * time.Second)
	if len(souinMock.GetAll()) != 2 {
		errors.GenerateError(t, "Souin API should have a record")
	}
	time.Sleep(10 * time.Second)
	if len(souinMock.GetAll()) == 1 {
		errors.GenerateError(t, "Souin API shouldn't have a record")
	}
}

func TestSouinAPI_GetBasePath(t *testing.T) {
	souinMock := mockSouinAPI()
	if souinMock.GetBasePath() != "/souinbasepath" {
		errors.GenerateError(t, "Souin API should be enabled")
	}
}

func TestSouinAPI_IsEnabled(t *testing.T) {
	souinMock := mockSouinAPI()
	if !souinMock.IsEnabled() {
		errors.GenerateError(t, "Souin API should be enabled")
	}
}

func TestSouinAPI_listKeys(t *testing.T) {
	souinMock := mockSouinAPI()
	souinMock.provider.Set("FIRST_KEY", []byte("Something"), configurationtypes.URL{}, time.Second)
	souinMock.provider.Set("SECOND_KEY", []byte("Something"), configurationtypes.URL{}, time.Second)
	souinMock.provider.Set("NOT_MATCH", []byte("Something"), configurationtypes.URL{}, time.Second)
	souinMock.provider.Set("NOT_MATCH_KEY_SUFFIX", []byte("Something"), configurationtypes.URL{}, time.Second)

	if len(souinMock.listKeys("(error")) != 0 {
		t.Error("An invalid regexp must return an empty list.")
	}

	keys := souinMock.listKeys(".+_KEY$")

	if len(keys) != 4 {
		t.Error("listKeys must return 4 items.")
	}
	if keys[0] != "FIRST_KEY" {
		t.Error("The first key must be equal to FIRST_KEY.")
	}
	if keys[1] != "SECOND_KEY" {
		t.Error("The second key must be equal to SECOND_KEY.")
	}
	if keys[2] != "STALE_FIRST_KEY" {
		t.Error("The third key must be equal to STALE_FIRST_KEY.")
	}
	if keys[3] != "STALE_SECOND_KEY" {
		t.Error("The fourth key must be equal to STALE_SECOND_KEY.")
	}
}

type mockSurrogateStorageError struct {
	*surrogate_providers.SouinSurrogateStorage
}

func (*mockSurrogateStorageError) Destruct() error {
	return fmt.Errorf("errored")
}

func TestSouinAPI_HandleRequest(t *testing.T) {
	souinMock := mockSouinAPI()
	souinMock.surrogateStorage.Destruct()
	souinMock.provider.DeleteMany(".+")
	souinMock.security = &auth.SecurityAPI{}

	req := httptest.NewRequest(http.MethodGet, "/souin-api/souinbasepath", nil)
	res := httptest.NewRecorder()

	souinMock.HandleRequest(res, req)
	b, _ := ioutil.ReadAll(res.Result().Body)
	if string(b) != "" {
		t.Error("The response body must be empty due to invalid token.")
	}
	if res.Result().StatusCode != http.StatusUnauthorized {
		t.Error("The response status code must be unauthorized.")
	}

	sr := res.Result()
	sr.Header.Set("Surrogate-Key", "THE_KEY")
	souinMock.provider.Set("first/key/stored", []byte("value"), configurationtypes.URL{}, time.Second)
	souinMock.surrogateStorage.Store(sr, "first/key/stored")
	souinMock.security = nil
	req = httptest.NewRequest(http.MethodGet, "/souin-api/souinbasepath/surrogate_keys", nil)
	res = httptest.NewRecorder()
	souinMock.HandleRequest(res, req)

	b, _ = ioutil.ReadAll(res.Result().Body)
	var surrogates map[string]string
	_ = json.Unmarshal(b, &surrogates)

	if len(surrogates) != 2 {
		t.Error("The surrogate storage must have 2 keys.")
	}
	if _, ok := surrogates["THE_KEY"]; !ok {
		t.Error("The key THE_KEY must exist in the surrogate storage.")
	}
	if _, ok := surrogates["STALE_THE_KEY"]; !ok {
		t.Error("The key STALE_THE_KEY must exist in the surrogate storage.")
	}

	req = httptest.NewRequest(http.MethodGet, "/souin-api/souinbasepath/inexistent_key", nil)
	res = httptest.NewRecorder()
	souinMock.HandleRequest(res, req)

	if res.Result().StatusCode != http.StatusNotFound {
		t.Error("The endpoint must return an HTTP not found response if the key doesn't exist.")
	}

	req = httptest.NewRequest(http.MethodGet, "/souin-api/souinbasepath", nil)
	res = httptest.NewRecorder()
	souinMock.HandleRequest(res, req)
	b, _ = ioutil.ReadAll(res.Result().Body)
	var values []string
	_ = json.Unmarshal(b, &values)

	if len(values) != 2 {
		t.Error("The keys first/key/stored and STALE_first/key/stored must exist in the storage provider.")
	}

	req = httptest.NewRequest("PURGE", "/souin-api/souinbasepath", nil)
	req.Header.Set("Surrogate-key", "THE_KEY")
	res = httptest.NewRecorder()
	souinMock.HandleRequest(res, req)

	if res.Result().StatusCode != http.StatusNoContent {
		t.Error("The endpoint must return a no content HTTP response.")
	}

	req = httptest.NewRequest("PURGE", "/souin-api/souinbasepath/flush", nil)
	res = httptest.NewRecorder()
	souinMock.surrogateStorage = &mockSurrogateStorageError{SouinSurrogateStorage: souinMock.surrogateStorage.(*surrogate_providers.SouinSurrogateStorage)}
	souinMock.HandleRequest(res, req)

	if res.Result().StatusCode != http.StatusNoContent {
		t.Error("The flush must return a no content HTTP response.")
	}

	if len(souinMock.provider.ListKeys()) != 0 {
		t.Error("The provider must not contains keys anymore after a full flush.")
	}
	if len(souinMock.surrogateStorage.List()) == 0 {
		t.Error("The surrogate storage must not contains keys anymore after a full flush error.")
	}

	res = httptest.NewRecorder()
	souinMock.surrogateStorage = souinMock.surrogateStorage.(*mockSurrogateStorageError).SouinSurrogateStorage
	souinMock.HandleRequest(res, req)

	if res.Result().StatusCode != http.StatusNoContent {
		t.Error("The flush must return a no content HTTP response.")
	}

	if len(souinMock.provider.ListKeys()) != 0 {
		t.Error("The provider must not contains keys anymore after a full flush.")
	}
	if len(souinMock.surrogateStorage.List()) != 0 {
		t.Error("The surrogate storage must not contains keys anymore after a full flush.")
	}

	req = httptest.NewRequest("PURGE", "/souin-api/souinbasepath/one_key", nil)
	res = httptest.NewRecorder()
	souinMock.HandleRequest(res, req)

	if res.Result().StatusCode != http.StatusNoContent {
		t.Error("The endpoint must return a no content HTTP response.")
	}

	req = httptest.NewRequest(http.MethodPost, "/souin-api/souinbasepath", nil)
	res = httptest.NewRecorder()
	souinMock.HandleRequest(res, req)
}
