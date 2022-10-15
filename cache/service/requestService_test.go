package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	souintypes "github.com/darkweak/souin/plugins/souin/types"
	"github.com/darkweak/souin/tests"

	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/errors"
)

func populateProviderWithFakeData(provider types.AbstractProviderInterface) {
	_ = provider.Set(tests.DOMAIN+tests.PATH, []byte("testing value is here for "+tests.PATH), tests.GetMatchedURL(tests.DOMAIN+tests.PATH), time.Duration(20)*time.Second)
	for i := 0; i < 25; i++ {
		_ = provider.Set(
			fmt.Sprintf("%s%s/%d", tests.DOMAIN, tests.PATH, i),
			[]byte(fmt.Sprintf("testing value is here for my first init of %s/%d", tests.PATH, i)),
			tests.GetMatchedURL(tests.DOMAIN+tests.PATH),
			time.Duration(20)*time.Second,
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
		Body:             io.NopCloser(strings.NewReader(body)),
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
				Host:       tests.DOMAIN,
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
			Host:             tests.DOMAIN,
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

func getKeyFromResponse(resp *http.Response, u configurationtypes.URL) string {
	headers := ""
	if u.Headers != nil && len(u.Headers) > 0 {
		for _, h := range u.Headers {
			headers += strings.ReplaceAll(resp.Request.Header.Get(h), " ", "")
		}
	}
	return resp.Request.Host + resp.Request.URL.Path + headers
}

func TestGetKeyFromResponse(t *testing.T) {
	resp := getKeyFromResponse(mockResponse(tests.PATH, http.MethodGet, "", 200), tests.GetMatchedURL(tests.PATH))
	urlCollapsed := tests.DOMAIN + tests.PATH
	if urlCollapsed != resp {
		errors.GenerateError(t, fmt.Sprintf("Key doesn't return %s but %s", urlCollapsed, resp))
	}
}

func shouldNotHaveKey(pathname string, pr types.AbstractProviderInterface) bool {
	return 0 >= len(pr.Get(pathname))
}

func verifyKeysExists(t *testing.T, path string, keys []string, isKeyDeleted bool, pr types.AbstractProviderInterface) {
	time.Sleep(10 * time.Second)

	for _, i := range keys {
		if !shouldNotHaveKey(tests.PATH+i, pr) == isKeyDeleted {
			errors.GenerateError(t, "The key "+tests.DOMAIN+path+i+" shouldn't exist.")
		}
	}
}

func TestKeyShouldBeDeletedOnPut(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	populateProviderWithFakeData(prs)
	mockResponse("/1", http.MethodPut, "My second response", 200)

	verifyKeysExists(t, tests.PATH, []string{"", "/1"}, true, prs)
}

func TestKeyShouldBeDeletedOnDelete(t *testing.T) {
	c := tests.MockConfiguration(tests.BaseConfiguration)
	prs := providers.InitializeProvider(c)
	populateProviderWithFakeData(prs)
	mockResponse("/1", http.MethodDelete, "", 200)
	verifyKeysExists(t, tests.PATH, []string{"", "/1"}, true, prs)
}

func TestRequestReverseProxy(t *testing.T) {
	request := httptest.NewRequest("GET", "http://"+tests.DOMAIN+tests.PATH, nil)
	conf := tests.MockConfiguration(tests.BaseConfiguration)
	u, _ := url.Parse(conf.GetReverseProxyURL())
	var excludedRegexp *regexp.Regexp = nil
	if conf.GetDefaultCache().GetRegex().Exclude != "" {
		excludedRegexp = regexp.MustCompile(conf.GetDefaultCache().GetRegex().Exclude)
	}
	response := RequestReverseProxy(
		request,
		souintypes.SouinRetrieverResponseProperties{
			RetrieverResponseProperties: types.RetrieverResponseProperties{
				Provider:      providers.InitializeProvider(conf),
				Configuration: conf,
				MatchedURL:    tests.GetMatchedURL(tests.PATH),
				ExcludeRegex:  excludedRegexp,
			},
			ReverseProxyURL: u,
		},
	)

	if response.Proxy == nil || response.Request == nil {
		errors.GenerateError(t, "Response proxy and request shouldn't be empty")
	}
}
