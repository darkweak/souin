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
)

const DOMAIN = "domain.com"
const PATH = "/testing"

func mockRedis() *redis.Client {
	return redisClientConnectionFactory()
}

func mockResponse(path string, method string, body string, code int) *http.Response {
	return &http.Response{
		Status:           "",
		StatusCode:       code,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           nil,
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
		generateError(t, fmt.Sprintf("Key doesn't return %s but %s", urlCollapsed, resp))
	}
}

func shouldNotHaveKey(pathname string) bool {
	client := redisClientConnectionFactory()
	_, err := client.Get(DOMAIN + pathname).Result()

	return err == redis.Nil
}

func TestKeyShouldBeDeletedOnPost(t *testing.T) {
	populateRedisWithFakeData()
	rewriteBody(mockResponse(PATH, http.MethodPost, "My second response", 201), mockRedis())
	time.Sleep(10 * time.Second)
	if !shouldNotHaveKey(PATH) {
		generateError(t, "The key "+DOMAIN+PATH+" shouldn't exist.")
	}
}

func verifyKeysExists(t *testing.T, path string, keys []string) {
	time.Sleep(10 * time.Second)

	for _, i := range keys {
		if !shouldNotHaveKey(PATH + i) {
			generateError(t, "The key "+DOMAIN+path+i+" shouldn't exist.")
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
