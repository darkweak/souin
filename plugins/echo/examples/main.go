package main

import (
	"fmt"
	"net/http"

	souin_echo "github.com/darkweak/souin/plugins/echo"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	// Use the Souin default configuration
	s := souin_echo.New(souin_echo.DevDefaultConfiguration)
	e.Use(s.Process)

	// Handler
	e.GET("/*", func(c echo.Context) error {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)

		c.Response().Write([]byte(fmt.Sprintf("<html><body><h1>%s%s</h1></body></html>", c.Request().Host, c.Request().URL.Path)))

		return nil
	})

	e.Logger.Fatal(e.Start(":80"))
}
