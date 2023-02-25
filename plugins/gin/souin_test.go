package gin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darkweak/souin/pkg/middleware"
	"github.com/gin-gonic/gin"
)

func Test_New(t *testing.T) {
	s := New(DevDefaultConfiguration)
	if s.SouinBaseHandler.Storer == nil {
		t.Error("The storer must be set.")
	}
	c := middleware.BaseConfiguration{}
	defer func() {
		if recover() == nil {
			t.Error("The New method must crash if an incomplete configuration is provided.")
		}
	}()
	New(c)
}

func prepare() (res *httptest.ResponseRecorder, res2 *httptest.ResponseRecorder, c *gin.Context, r *gin.Engine) {
	req := httptest.NewRequest(http.MethodGet, "/handled", nil)
	req.Header = http.Header{}
	res = httptest.NewRecorder()
	res2 = httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	s := New(DevDefaultConfiguration)
	c, r = gin.CreateTestContext(res)
	c.Request = req
	r.Use(s.Process())
	return
}

func Test_SouinGinPlugin_Process(t *testing.T) {
	res, res2, c, r := prepare()
	r.GET("/handled", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})
	r.ServeHTTP(res, c.Request)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the stored directive.")
	}

	r.ServeHTTP(res2, c.Request)
	if res2.Result().Header.Get("Cache-Status") != "Souin; hit; ttl=4; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header with the hit and ttl directives.")
	}
	if res2.Result().Header.Get("Age") != "1" {
		t.Error("The response must contain a Age header with the value 1.")
	}
}

func Test_SouinGinPlugin_Process_CannotHandle(t *testing.T) {
	res, res2, c, r := prepare()
	c.Request.Header.Set("Cache-Control", "no-cache")
	r.GET("/handled", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})
	r.ServeHTTP(res, c.Request)

	if res.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}

	r.ServeHTTP(res2, c.Request)
	if res2.Result().Header.Get("Cache-Status") != "Souin; fwd=uri-miss; stored; key=GET-example.com-/handled" {
		t.Error("The response must contain a Cache-Status header without the stored directive and with the uri-miss only.")
	}
	if res2.Result().Header.Get("Age") != "" {
		t.Error("The response must not contain a Age header.")
	}
}
