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

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/configuration"
	"regexp"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/providers"
	"log"
	"github.com/darkweak/souin/configuration_types"
)

func MockConfiguration() configuration_types.AbstractConfigurationInterface {
	var config configuration.Configuration
	e := config.Parse([]byte(`
default_cache:
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000
reverse_proxy_url: 'http://traefik'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50
    headers:
      - Authorization
      - 'Content-Type'
`))
	if e != nil {
		log.Fatal(e)
	}
	return &config
}

const DOMAIN = "domain.com"
const PATH = "/testing"

func MockInitializeRegexp(configurationInstance configuration_types.AbstractConfigurationInterface) regexp.Regexp {
	u := ""
	for k := range configurationInstance.GetUrls() {
		if "" != u {
			u += "|"
		}
		u += "(" + k + ")"
	}

	return *regexp.MustCompile(u)
}

func getMatchedURL(key string) configuration_types.URL {
	config := MockConfiguration()
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configuration_types.URL{
		TTL:     config.GetDefaultCache().TTL,
		Headers: config.GetDefaultCache().Headers,
	}
	if "" != regexpURL {
		matchedURL = config.GetUrls()[regexpURL]
	}

	return matchedURL
}

func populateProviderWithFakeData(provider types.AbstractProviderInterface) {
	basePath := "/testing"
	domain := "domain.com"

	provider.SetRequestInCache(domain+basePath, []byte("testing value is here for "+basePath), getMatchedURL(domain+basePath))
	for i := 0; i < 25; i++ {
		provider.SetRequestInCache(
			fmt.Sprintf("%s%s/%d", domain, basePath, i),
			[]byte(fmt.Sprintf("testing value is here for my first init of %s/%d", basePath, i)),
			getMatchedURL(domain+basePath),
		)
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

func shouldNotHaveKey(pathname string, pr types.AbstractProviderInterface) bool {
	r := pr.GetRequestInCache(pathname)
	if 0 < len(r.Response) {
		return false
	}

	return true
}

func mockRewriteBody(method string, body string, path string, code int, pr types.AbstractProviderInterface) error {
	config := MockConfiguration()
	return rewriteBody(
		mockResponse(PATH+path, method, body, code),
		&types.RetrieverResponseProperties{
			Configuration: config,
			Provider:      pr,
			MatchedURL:    getMatchedURL(DOMAIN + PATH + path),
		},
	)
}

func TestKeyShouldBeDeletedOnPost(t *testing.T) {
	c := MockConfiguration()
	prs := providers.InitializeProvider(c)
	populateProviderWithFakeData(prs)
	mockRewriteBody(http.MethodPost, "My second response", "/1", 201, prs)
	time.Sleep(10 * time.Second)
	if !shouldNotHaveKey(PATH, prs) {
		errors.GenerateError(t, "The key "+DOMAIN+PATH+" shouldn't exist.")
	}
}

func TestRewriteBody(t *testing.T) {
	c := MockConfiguration()
	prs := providers.InitializeProvider(c)
	err := mockRewriteBody(http.MethodPost, "My second response", "", 201, prs)
	if err != nil {
		errors.GenerateError(t, "Rewrite body can't return errors")
	}
}

func verifyKeysExists(t *testing.T, path string, keys []string, isKeyDeleted bool, pr types.AbstractProviderInterface) {
	time.Sleep(10 * time.Second)

	for _, i := range keys {
		if !shouldNotHaveKey(PATH+i, pr) == isKeyDeleted {
			errors.GenerateError(t, "The key "+DOMAIN+path+i+" shouldn't exist.")
		}
	}
}

func TestKeyShouldBeDeletedOnPut(t *testing.T) {
	c := MockConfiguration()
	prs := providers.InitializeProvider(c)
	populateProviderWithFakeData(prs)
	mockResponse("/1", http.MethodPut, "My second response", 200)

	verifyKeysExists(t, PATH, []string{"", "/1"}, true, prs)
}

func TestKeyShouldBeDeletedOnDelete(t *testing.T) {
	c := MockConfiguration()
	prs := providers.InitializeProvider(c)
	populateProviderWithFakeData(prs)
	mockResponse("/1", http.MethodDelete, "", 200)
	verifyKeysExists(t, PATH, []string{"", "/1"}, true, prs)
}

func TestRequestReverseProxy(t *testing.T) {
	request, _ := http.NewRequest(http.MethodGet, "http://localhost", nil)
	conf := MockConfiguration()
	response := RequestReverseProxy(
		request,
		request.URL,
		&types.RetrieverResponseProperties{
			Provider:      providers.InitializeProvider(conf),
			Configuration: conf,
			MatchedURL:    getMatchedURL(PATH),
		},
	)

	if string(response.Response) != "bad" {
		errors.GenerateError(t, "Response should be bad due to no host available")
	}

	if response.Proxy == nil || response.Request == nil {
		errors.GenerateError(t, "Response proxy and request shouldn't be empty")
	}
}

func TestCommonLoadingRequest(t *testing.T) {
	body := "My testable response"
	lenBody := len([]byte(body))
	response := commonLoadingRequest(mockResponse(PATH, http.MethodGet, body, 200))

	if "" == string(response) {
		errors.GenerateError(t, "Body shouldn't be empty")
	}
	if body != string(response) || lenBody != len(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}

	body = "Another body with <h1>HTML</h1>"
	lenBody = len([]byte(body))
	response = commonLoadingRequest(mockResponse(PATH, http.MethodGet, body, 200))

	if "" == string(response) {
		errors.GenerateError(t, "Body shouldn't be empty")
	}
	if body != string(response) || lenBody != len(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}

	response = commonLoadingRequest(mockResponse(PATH+"/another", http.MethodGet, body, 200))

	if "" == string(response) {
		errors.GenerateError(t, "Body shouldn't be empty")
	}
	if body != string(response) || lenBody != len(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}
}
