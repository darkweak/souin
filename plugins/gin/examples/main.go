package main

import (
	"net/http"

	souin_gin "github.com/darkweak/souin/plugins/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	// Use the Souin default configuration
	s := souin_gin.New(souin_gin.DevDefaultConfiguration)
	r.Use(s.Process())

	// Handler
	r.GET("/default", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})
	r.GET("/excluded", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, Excluded!")
	})

	// Start server
	_ = r.Run(":80")
}
