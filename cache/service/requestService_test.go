package service

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/errors"
	"log"
	"regexp"
)

func MockConfiguration() configurationtypes.AbstractConfigurationInterface {
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

func MockInitializeRegexp(configurationInstance configurationtypes.AbstractConfigurationInterface) regexp.Regexp {
	u := ""
	for k := range configurationInstance.GetUrls() {
		if "" != u {
			u += "|"
		}
		u += "(" + k + ")"
	}

	return *regexp.MustCompile(u)
}

func getMatchedURL(key string) configurationtypes.URL {
	config := MockConfiguration()
	regexpUrls := MockInitializeRegexp(config)
	regexpURL := regexpUrls.FindString(key)
	matchedURL := configurationtypes.URL{
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

	provider.Set(domain+basePath, []byte("testing value is here for "+basePath), getMatchedURL(domain+basePath), time.Duration(20) * time.Second)
	for i := 0; i < 25; i++ {
		provider.Set(
			fmt.Sprintf("%s%s/%d", domain, basePath, i),
			[]byte(fmt.Sprintf("testing value is here for my first init of %s/%d", basePath, i)),
			getMatchedURL(domain+basePath),
			time.Duration(20) * time.Second,
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
		Body:             ioutil.NopCloser(strings.NewReader(body)),
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
	r := pr.Get(pathname)
	if 0 < len(r) {
		return false
	}

	return true
}

func mockRewriteResponse(method string, body string, path string, code int) []byte {
	return RewriteResponse(mockResponse(PATH+path, method, body, code))
}

func TestKeyShouldBeDeletedOnPost(t *testing.T) {
	c := MockConfiguration()
	prs := providers.InitializeProvider(c)
	populateProviderWithFakeData(prs)
	res := mockRewriteResponse(http.MethodPost, "My second response", "/1", 201)
	if len(res) <= 0 {
		errors.GenerateError(t, "The response should be valid and filled")
	}
	time.Sleep(10 * time.Second)
	if !shouldNotHaveKey(PATH, prs) {
		errors.GenerateError(t, "The key "+DOMAIN+PATH+" shouldn't exist.")
	}
}

func TestRewriteBody(t *testing.T) {
	response := mockRewriteResponse(http.MethodPost, "My second response", "", 201)
	if response == nil || len(response) <= 0 {
		errors.GenerateError(t, "Rewrite body should return an empty response")
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
	u, _ := url.Parse(conf.GetReverseProxyURL())
	response := RequestReverseProxy(
		request,
		&types.RetrieverResponseProperties{
			Provider:      providers.InitializeProvider(conf),
			Configuration: conf,
			MatchedURL:    getMatchedURL(PATH),
			ReverseProxyURL: u,
		},
	)

	if response.Proxy == nil || response.Request == nil {
		errors.GenerateError(t, "Response proxy and request shouldn't be empty")
	}
}

func TestCommonLoadingRequest(t *testing.T) {
	body := "My testable response"
	lenBody := len([]byte(body))
	response := responseBodyExtractor(mockResponse(PATH, http.MethodGet, body, 200))

	if "" == string(response) {
		errors.GenerateError(t, "Body shouldn't be empty")
	}
	if body != string(response) || lenBody != len(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}

	body = "Another body with <h1>HTML</h1>"
	lenBody = len([]byte(body))
	response = responseBodyExtractor(mockResponse(PATH, http.MethodGet, body, 200))

	if "" == string(response) {
		errors.GenerateError(t, "Body shouldn't be empty")
	}
	if body != string(response) || lenBody != len(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}

	response = responseBodyExtractor(mockResponse(PATH+"/another", http.MethodGet, body, 200))

	if "" == string(response) {
		errors.GenerateError(t, "Body shouldn't be empty")
	}
	if body != string(response) || lenBody != len(response) {
		errors.GenerateError(t, fmt.Sprintf("Body %s doesn't match attempted %s", string(response), body))
	}
}
