package cache

import (
	"testing"
	"net/http"
	"net/url"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"github.com/go-redis/redis"
	"time"
	"github.com/darkweak/souin/cache/providers"
)

const DOMAIN = "domain.com"
const PATH = "/testing"

func populateRedisWithFakeData() {
	client := cache.RedisConnectionFactory()
	duration := time.Duration(120) * time.Second
	basePath := "/testing"
	domain := "domain.com"

	client.Set(domain+basePath, "testing value is here for "+basePath, duration)
	for i := 0; i < 25; i++ {
		client.Set(domain+basePath+"/"+string(i), "testing value is here for my first init of "+basePath+"/"+string(i), duration)
	}
}

func mockRedis() *cache.Redis {
	return cache.RedisConnectionFactory()
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
		GenerateError(t, fmt.Sprintf("Key doesn't return %s but %s", urlCollapsed, resp))
	}
}

func shouldNotHaveKey(pathname string) bool {
	client := cache.RedisConnectionFactory()
	_, err := client.Get(DOMAIN + pathname).Result()

	return err == redis.Nil
}

func TestKeyShouldBeDeletedOnPost(t *testing.T) {
	populateRedisWithFakeData()
	rewriteBody(mockResponse(PATH, http.MethodPost, "My second response", 201), mockRedis())
	time.Sleep(10 * time.Second)
	if !shouldNotHaveKey(PATH) {
		GenerateError(t, "The key "+DOMAIN+PATH+" shouldn't exist.")
	}
}

func verifyKeysExists(t *testing.T, path string, keys []string) {
	time.Sleep(10 * time.Second)

	for _, i := range keys {
		if !shouldNotHaveKey(PATH + i) {
			GenerateError(t, "The key "+DOMAIN+path+i+" shouldn't exist.")
		}
	}
}

func TestKeyShouldBeDeletedOnPut(t *testing.T) {
	populateRedisWithFakeData()
	rewriteBody(mockResponse(PATH+"/1", http.MethodPut, "My second response", 200), mockRedis())
	verifyKeysExists(t, PATH, []string{"", "/1"})
}

func TestKeyShouldBeDeletedOnDelete(t *testing.T) {
	populateRedisWithFakeData()
	rewriteBody(mockResponse(PATH+"/1", http.MethodDelete, "", 200), mockRedis())
	verifyKeysExists(t, PATH, []string{"", "/1"})
}
