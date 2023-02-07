package main

import (
	"net/http"

	souin_echo "github.com/darkweak/souin/plugins/echo"
	"github.com/labstack/echo/v4"
)

func main() {
	e := echo.New()

	// iterator := 0

	// Use the Souin default configuration
	s := souin_echo.NewMiddleware(souin_echo.DevDefaultConfiguration)
	e.Use(s.Process)

	// Handler
	e.GET("/*", func(c echo.Context) error {
		// if iterator > 2 {
		// 	c.String(http.StatusInternalServerError, "Internal Server Error")
		// }
		// iterator++
		// c.Response().Header().Set("Cache-Control", "stale-if-error=200")
		return c.String(http.StatusOK, "Hello, World!")
	})

	e.Logger.Fatal(e.Start(":80"))
}
