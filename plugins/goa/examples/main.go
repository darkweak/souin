package main

import (
	"net/http"

	httpcache "github.com/darkweak/souin/plugins/goa"
	goahttp "goa.design/goa/v3/http"
)

func defaultHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}

func main() {
	// Use the Souin default configuration
	g := goahttp.NewMuxer()
	g.Use(httpcache.NewHTTPCache(httpcache.DevDefaultConfiguration))

	g.Handle(http.MethodGet, "/excluded", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, Excluded!"))
	})
	g.Handle(http.MethodGet, "/{path}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	http.ListenAndServe(":80", g)
}
