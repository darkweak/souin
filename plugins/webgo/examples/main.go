package main

import (
	"net/http"
	"time"

	"github.com/bnkamalesh/webgo/v6"
	cache "github.com/darkweak/souin/plugins/webgo"
)

func defaultHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello, World!"))
}

func getRoutes() []*webgo.Route {
	return []*webgo.Route{
		{
			Name:          "default",
			Method:        http.MethodGet,
			Pattern:       "/:all*",
			Handlers:      []http.HandlerFunc{defaultHandler},
			TrailingSlash: true,
		},
	}
}

func main() {
	cfg := &webgo.Config{
		Host:         "",
		Port:         "80",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 1 * time.Hour,
	}

	// Use the Souin default configuration
	httpcache := cache.NewHTTPCache(cache.DevDefaultConfiguration)
	router := webgo.NewRouter(cfg, getRoutes()...)
	router.Use(httpcache.Middleware)
	router.Start()
}
