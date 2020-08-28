package service

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/configuration"
	"regexp"
)

const DOMAIN = "domain.com"
const PATH = "/testing"

func MockInitializeRegexp(configurationInstance configuration.Configuration) regexp.Regexp {
	u := ""
	for k := range configurationInstance.URLs {
		if "" != u {
			u += "|"
		}
		u += "(" + k + ")"
	}

	return *regexp.MustCompile(u)
}

func getMatchedURL(key string) configuration.URL {
	config := configuration.GetConfig()
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configuration.URL{
		TTL:       config.DefaultCache.TTL,
		Providers: config.DefaultCache.Providers,
		Headers:   config.DefaultCache.Headers,
	}
	if "" != regexpURL {
		matchedURL = config.URLs[regexpURL]
	}

	return matchedURL
}

func populateProvidersWithFakeData() {
	basePath := "/testing"
	domain := "domain.com"

	for _, provider := range map[string]providers.AbstractProviderInterface{"redis": mockRedis(), "memory": mockMemory()} {
		provider.SetRequestInCache(domain+basePath, []byte("testing value is here for "+basePath), getMatchedURL(domain+basePath))
		for i := 0; i < 25; i++ {
			provider.SetRequestInCache(domain+basePath+"/"+string(i), []byte("testing value is here for my first init of "+basePath+"/"+string(i)), getMatchedURL(domain+basePath))
		}
	}
}

func mockRedis() *providers.Redis {
	return providers.RedisConnectionFactory(configuration.GetConfig())
}

func mockMemory() providers.Memory {
	return providers.MemoryConnectionFactory(configuration.GetConfig())
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
	resp := getKeyFromResponse(mockResponse(PATH, http.MethodGet, "", 200), getMatchedURL(PATH))
	urlCollapsed := DOMAIN + PATH
	if urlCollapsed != resp {
		errors.GenerateError(t, fmt.Sprintf("Key doesn't return %s but %s", urlCollapsed, resp))
	}
}

func shouldNotHaveKey(pathname string) bool {
	config := configuration.GetConfig()
	redisClient := providers.RedisConnectionFactory(config)
	_, redisErr := redisClient.Get(redisClient.Context(), DOMAIN+pathname).Result()
	memoryClient := providers.MemoryConnectionFactory(config)
	_, memoryErr := memoryClient.Get(DOMAIN + pathname)

	return memoryErr != nil && redisErr != nil
}

func mockRewriteBody (method string, body string, path string, code int) error {
	config := configuration.GetConfig()
	return rewriteBody(mockResponse(PATH + path, method, body, code), map[string]providers.AbstractProviderInterface{"redis": mockRedis(), "memory": mockMemory()}, config, getMatchedURL(DOMAIN+PATH+path))
}

func TestKeyShouldBeDeletedOnPost(t *testing.T) {
	populateProvidersWithFakeData()
	mockRewriteBody("", http.MethodPost, "My second response", 201)
	time.Sleep(10 * time.Second)
	if !shouldNotHaveKey(PATH) {
		errors.GenerateError(t, "The key "+DOMAIN+PATH+" shouldn't exist.")
	}
}

func TestRewriteBody(t *testing.T) {
	err := mockRewriteBody("", http.MethodPost, "My second response", 201)
	if err != nil {
		errors.GenerateError(t, "Rewrite body can't return errors")
	}
}

func verifyKeysExists(t *testing.T, path string, keys []string, isKeyDeleted bool) {
	time.Sleep(10 * time.Second)

	for _, i := range keys {
		if !shouldNotHaveKey(PATH + i) == isKeyDeleted {
			errors.GenerateError(t, "The key "+DOMAIN+path+i+" shouldn't exist.")
		}
	}
}

func TestKeyShouldBeDeletedOnPut(t *testing.T) {
	populateProvidersWithFakeData()
	mockResponse("/1", http.MethodPut, "My second response", 200)

	verifyKeysExists(t, PATH, []string{"", "/1"}, true)
}

func TestKeyShouldBeDeletedOnDelete(t *testing.T) {
	populateProvidersWithFakeData()
	mockResponse("/1", http.MethodDelete, "", 200)
	verifyKeysExists(t, PATH, []string{"", "/1"}, true)
}

func TestRequestReverseProxy(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "http://localhost", nil)
	conf := configuration.GetConfig()
	response := RequestReverseProxy(request, request.URL, *providers.InitializeProviders(conf), conf, getMatchedURL(PATH))

	if response.Response != "bad" {
		errors.GenerateError(t, "Response should be bad due to no host available")
	}

	if response.Proxy == nil || response.Request == nil {
		errors.GenerateError(t, "Response proxy and request shouldn't be empty")
	}
}

func TestCommonLoadingRequest(t *testing.T)  {
	body := "My testable response"
	response := commonLoadingRequest(mockResponse(PATH, http.MethodGet, body, 200))

	if body != string(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}

	body = "Another body with <h1>HTML</h1>"
	response = commonLoadingRequest(mockResponse(PATH, http.MethodGet, body, 200))

	if body != string(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}

	response = commonLoadingRequest(mockResponse(PATH+"/another", http.MethodGet, body, 200))

	if body != string(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}
}
