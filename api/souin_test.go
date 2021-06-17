package api

import (
	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"testing"
	"time"
)

func mockSouinAPI() *SouinAPI {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(config)
	security := auth.InitializeSecurity(config)
	return &SouinAPI{
		"/souinbasepath",
		true,
		prs,
		security,
		ykeys.InitializeYKeys(config.Ykeys),
	}
}

func TestSouinAPI_BulkDelete(t *testing.T) {
	souinMock := mockSouinAPI()
	souinMock.provider.Set("firstKey", []byte("value"), tests.GetMatchedURL("firstKey"), 20*time.Second)
	souinMock.provider.Set("secondKey", []byte("value"), tests.GetMatchedURL("secondKey"), 20*time.Second)
	time.Sleep(3 * time.Second)
	if len(souinMock.GetAll()) != 2 {
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
	if len(souinMock.GetAll()) != 1 {
		errors.GenerateError(t, "Souin API should have a record")
	}
	souinMock.Delete("key")
	time.Sleep(3 * time.Second)
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
	if len(souinMock.GetAll()) != 1 {
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
