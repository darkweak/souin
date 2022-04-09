package main

import (
	"fmt"

	cache "github.com/darkweak/souin/plugins/fiber"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()
	app.Use(cache.NewHTTPCache(cache.DevDefaultConfiguration).Handle)

	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 👋!")
	})

	fmt.Println(app.Listen(":80"))
}
