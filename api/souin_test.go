package api

import (
	"github.com/darkweak/souin/api/auth"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"regexp"
	"testing"
	"time"
)

func mockSouinAPI() *SouinAPI {
	config := tests.MockConfiguration()
	prs := providers.InitializeProvider(config)
	security := auth.InitializeSecurity(config)
	return &SouinAPI{
		"/souinbasepath",
		true,
		prs,
		security,
	}
}

func TestSouinAPI_BulkDelete(t *testing.T) {
	souinMock := mockSouinAPI()
	for _, provider := range souinMock.providers {
		provider.Set("key", []byte("value"), tests.GetMatchedURL("key"), 20 * time.Second)
		provider.Set("key2", []byte("value"), tests.GetMatchedURL("key"), 20 * time.Second)
	}
	time.Sleep(3 * time.Second)
	for _, v := range souinMock.GetAll() {
		if len(v) != 2 {
			errors.GenerateError(t, "Souin API should have a record")
		}
	}
	souinMock.BulkDelete(regexp.MustCompile(".+"))
	time.Sleep(5 * time.Second)
	for _, v := range souinMock.GetAll() {
		if len(v) != 0 {
			errors.GenerateError(t, "Souin API should have a record")
		}
	}
}

func TestSouinAPI_Delete(t *testing.T) {
	souinMock := mockSouinAPI()
	for _, provider := range souinMock.providers {
		provider.Set("key", []byte("value"), tests.GetMatchedURL("key"), 20 * time.Second)
	}
	time.Sleep(3 * time.Second)
	for _, v := range souinMock.GetAll() {
		if len(v) != 1 {
			errors.GenerateError(t, "Souin API should have a record")
		}
	}
	souinMock.Delete("key")
	time.Sleep(3 * time.Second)
	for _, v := range souinMock.GetAll() {
		if len(v) == 1 {
			errors.GenerateError(t, "Souin API shouldn't have a record")
		}
	}
}

func TestSouinAPI_GetAll(t *testing.T) {
	souinMock := mockSouinAPI()
	for _, v := range souinMock.GetAll() {
		if len(v) > 0 {
			errors.GenerateError(t, "Souin API shouldn't have a record")
		}
	}

	for _, provider := range souinMock.providers {
		provider.Set("key", []byte("value"), tests.GetMatchedURL("key"), 6 * time.Second)
	}
	time.Sleep(3 * time.Second)
	for _, v := range souinMock.GetAll() {
		if len(v) != 1 {
			errors.GenerateError(t, "Souin API should have a record")
		}
	}
	souinMock.providers["redis"].Delete("key")
	time.Sleep(10 * time.Second)
	for _, v := range souinMock.GetAll() {
		if len(v) == 1 {
			errors.GenerateError(t, "Souin API shouldn't have a record")
		}
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
