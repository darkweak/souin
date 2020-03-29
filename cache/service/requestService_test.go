package service

import (
	"fmt"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/errors"
	"github.com/go-redis/redis"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

const DOMAIN = "domain.com"
const PATH = "/testing"

func populateRedisWithFakeData() {
	client := providers.RedisConnectionFactory()
	duration := time.Duration(120) * time.Second
	basePath := "/testing"
	domain := "domain.com"

	client.Set(domain+basePath, "testing value is here for "+basePath, duration)
	for i := 0; i < 25; i++ {
		client.Set(domain+basePath+"/"+string(i), "testing value is here for my first init of "+basePath+"/"+string(i), duration)
	}
}

func mockRedis() *providers.Redis {
	return providers.RedisConnectionFactory()
}

func mockResponse(path string, method string, body string, code int) *http.Response {
	return &http.Response{
		Status:           "",
		StatusCode:       code,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           make(map[string][]string),
		Body:             io.ReadCloser(ioutil.NopCloser(strings.NewReader(body))),
		ContentLength:    0,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request: &http.Request{
			Method: method,
			URL: &url.URL{
				Scheme:     "",
				Opaque:     "",
				User:       nil,
				Host:       DOMAIN,
				Path:       path,
				RawPath:    "",
				ForceQuery: false,
				RawQuery:   "",
				Fragment:   "",
			},
			Proto:            "",
			ProtoMajor:       0,
			ProtoMinor:       0,
			Header:           nil,
			Body:             nil,
			GetBody:          nil,
			ContentLength:    0,
			TransferEncoding: nil,
			Close:            false,
			Host:             DOMAIN,
			Form:             nil,
			PostForm:         nil,
			MultipartForm:    nil,
			Trailer:          nil,
			RemoteAddr:       "",
			RequestURI:       "",
			TLS:              nil,
			Response:         nil,
		},
		TLS: nil,
	}
}

func TestGetKeyFromResponse(t *testing.T) {
	resp := getKeyFromResponse(mockResponse(PATH, http.MethodGet, "", 200))
	urlCollapsed := DOMAIN + PATH
	if urlCollapsed != resp {
		errors.GenerateError(t, fmt.Sprintf("Key doesn't return %s but %s", urlCollapsed, resp))
	}
}

func shouldNotHaveKey(pathname string) bool {
	client := providers.RedisConnectionFactory()
	_, err := client.Get(DOMAIN + pathname).Result()

	return err == redis.Nil
}

func TestKeyShouldBeDeletedOnPost(t *testing.T) {
	populateRedisWithFakeData()
	rewriteBody(mockResponse(PATH, http.MethodPost, "My second response", 201), []providers.AbstractProviderInterface{mockRedis()})
	time.Sleep(10 * time.Second)
	if !shouldNotHaveKey(PATH) {
		errors.GenerateError(t, "The key "+DOMAIN+PATH+" shouldn't exist.")
	}
}

func verifyKeysExists(t *testing.T, path string, keys []string) {
	time.Sleep(10 * time.Second)

	for _, i := range keys {
		if !shouldNotHaveKey(PATH + i) {
			errors.GenerateError(t, "The key "+DOMAIN+path+i+" shouldn't exist.")
		}
	}
}

func TestKeyShouldBeDeletedOnPut(t *testing.T) {
	populateRedisWithFakeData()
	rewriteBody(mockResponse(PATH+"/1", http.MethodPut, "My second response", 200), []providers.AbstractProviderInterface{mockRedis()})
	verifyKeysExists(t, PATH, []string{"", "/1"})
}

func TestKeyShouldBeDeletedOnDelete(t *testing.T) {
	populateRedisWithFakeData()
	rewriteBody(mockResponse(PATH+"/1", http.MethodDelete, "", 200), []providers.AbstractProviderInterface{mockRedis()})
	verifyKeysExists(t, PATH, []string{"", "/1"})
}
