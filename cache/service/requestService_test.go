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

func populateProvidersWithFakeData(ps map[string]providers.AbstractProviderInterface) {
	basePath := "/testing"
	domain := "domain.com"

	for _, provider := range ps {
		provider.SetRequestInCache(domain+basePath, []byte("testing value is here for "+basePath), getMatchedURL(domain+basePath))
		for i := 0; i < 25; i++ {
			provider.SetRequestInCache(domain+basePath+"/"+string(i), []byte("testing value is here for my first init of "+basePath+"/"+string(i)), getMatchedURL(domain+basePath))
		}
	}
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

func shouldNotHaveKey(pathname string, prs map[string]providers.AbstractProviderInterface) bool {
	for _, v := range prs {
		r := v.GetRequestInCache(pathname)
		if "" != r.Response {
			return false
		}
	}

	return true
}

func mockRewriteBody (method string, body string, path string, code int, prs map[string]providers.AbstractProviderInterface) error {
	config := configuration.GetConfig()
	return rewriteBody(mockResponse(PATH + path, method, body, code), prs, config, getMatchedURL(DOMAIN+PATH+path))
}

func TestKeyShouldBeDeletedOnPost(t *testing.T) {
	prs := providers.InitializeProviders(configuration.GetConfig())
	populateProvidersWithFakeData(prs)
	mockRewriteBody(http.MethodPost, "My second response", "/1", 201, prs)
	time.Sleep(10 * time.Second)
	if !shouldNotHaveKey(PATH, prs) {
		errors.GenerateError(t, "The key "+DOMAIN+PATH+" shouldn't exist.")
	}
}

func TestRewriteBody(t *testing.T) {
	prs := providers.InitializeProviders(configuration.GetConfig())
	err := mockRewriteBody(http.MethodPost, "My second response", "", 201, prs)
	if err != nil {
		errors.GenerateError(t, "Rewrite body can't return errors")
	}
}

func verifyKeysExists(t *testing.T, path string, keys []string, isKeyDeleted bool, prs map[string]providers.AbstractProviderInterface) {
	time.Sleep(10 * time.Second)

	for _, i := range keys {
		if !shouldNotHaveKey(PATH + i, prs) == isKeyDeleted {
			errors.GenerateError(t, "The key "+DOMAIN+path+i+" shouldn't exist.")
		}
	}
}

func TestKeyShouldBeDeletedOnPut(t *testing.T) {
	prs := providers.InitializeProviders(configuration.GetConfig())
	populateProvidersWithFakeData(prs)
	mockResponse("/1", http.MethodPut, "My second response", 200)

	verifyKeysExists(t, PATH, []string{"", "/1"}, true, prs)
}

func TestKeyShouldBeDeletedOnDelete(t *testing.T) {
	prs := providers.InitializeProviders(configuration.GetConfig())
	populateProvidersWithFakeData(prs)
	mockResponse("/1", http.MethodDelete, "", 200)
	verifyKeysExists(t, PATH, []string{"", "/1"}, true, prs)
}

func TestRequestReverseProxy(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "http://localhost", nil)
	conf := configuration.GetConfig()
	response := RequestReverseProxy(request, request.URL, providers.InitializeProviders(conf), conf, getMatchedURL(PATH))

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
