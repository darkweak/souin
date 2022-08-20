package main

import (
	"net/http"
	"time"

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
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		c.Response().WriteHeader(http.StatusOK)

		res := []byte(`{"a": 1}`)
		for i := 0; i < len(res); i++ {
			c.Response().Write([]byte{res[i]})
			time.Sleep(time.Second)
		}

		return nil
	})

	e.Logger.Fatal(e.Start(":80"))
}
