package main

import (
	"net/http"

	cache "github.com/darkweak/souin/plugins/chi"
	"github.com/go-chi/chi/v5"
)

func defaultHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}

func main() {
	// Use the Souin default configuration
	router := chi.NewRouter()
	router.Use(cache.NewHTTPCache(cache.DevDefaultConfiguration).Handle)
	router.Get("/*", defaultHandler)
	http.ListenAndServe(":80", router)
}
