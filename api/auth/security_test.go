package auth

import (
	"fmt"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/tests"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInitializeSecurity(t *testing.T) {
	var config configuration.Configuration
	_ = config.Parse([]byte(`
api:
  security:
    basepath: /mybasepath
    secret: your_secret_key
    enable: true
    users:
      - username: user1
        password: test
default_cache:
  headers:
    - Authorization
  port:
    web: 80
    tls: 443
  regex:
    exclude: 'ARegexHere'
  ttl: 1000
reverse_proxy_url: 'http://domain.com:81'
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
	security := InitializeSecurity(&config)
	if security.basePath != "/mybasepath" {
		errors.GenerateError(t, "Basepath should be /mybasepath")
	}

	security = InitializeSecurity(tests.MockConfiguration(tests.BaseConfiguration))
	if security.basePath != "/authentication" {
		errors.GenerateError(t, "Basepath should be /authentication")
	}
}

func TestSecurityAPI_GetBasePath(t *testing.T) {
	security := InitializeSecurity(tests.MockConfiguration(tests.BaseConfiguration))
	if security.GetBasePath() != "/authentication" {
		errors.GenerateError(t, "Basepath should be /authentication")
	}
}

func TestSecurityAPI_IsEnabled(t *testing.T) {
	security := InitializeSecurity(tests.MockConfiguration(tests.BaseConfiguration))
	if security.IsEnabled() != true {
		errors.GenerateError(t, "Security should be enabled")
	}
}

func TestSecurityAPI_HandleRequest(t *testing.T) {
	config := tests.MockConfiguration(tests.BaseConfiguration)
	security := InitializeSecurity(config)
	r := httptest.NewRequest("GET", fmt.Sprintf("http://%s%s/invalid_path", config.GetAPI().BasePath, security.basePath), nil)
	w := httptest.NewRecorder()

	security.HandleRequest(w, r)
	if w.Result().StatusCode != http.StatusOK {
		errors.GenerateError(t, "Status code should be 200")
	}
	b, _ := ioutil.ReadAll(w.Result().Body)
	if len(b) != 0 {
		errors.GenerateError(t, "Body should be an empty array")
	}

	r = httptest.NewRequest("POST", fmt.Sprintf("http://%s%s/invalid_path", config.GetAPI().BasePath, security.basePath), nil)
	w = httptest.NewRecorder()

	security.HandleRequest(w, r)
	if w.Result().StatusCode != http.StatusOK {
		errors.GenerateError(t, "Status code should be 200")
	}
	b, _ = ioutil.ReadAll(w.Result().Body)
	if len(b) != 0 {
		errors.GenerateError(t, "Body should be an empty array")
	}

	w = httptest.NewRecorder()
	http.SetCookie(w, tests.GetValidToken())
	r = &http.Request{
		Header: http.Header{
			"Cookie": w.HeaderMap["Set-Cookie"],
		},
	}
	security.HandleRequest(w, r)
}
