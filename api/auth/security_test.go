package auth

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/plugins/souin/configuration"
	"github.com/darkweak/souin/tests"
)

func TestInitializeSecurity(t *testing.T) {
	var config configuration.Configuration
	_ = config.Parse([]byte(`
api:
  security:
    basepath: /mybasepath
    secret: your_secret_key
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
  ttl: 1000s
reverse_proxy_url: 'http://domain.com:81'
ssl_providers:
  - traefik
urls:
  'domain.com/':
    ttl: 1000s
    headers:
      - Authorization
  'mysubdomain.domain.com':
    ttl: 50s
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
			"Cookie": w.Header()["Set-Cookie"],
		},
	}
	security.HandleRequest(w, r)
}
