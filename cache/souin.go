package cache

import (
	"net/http"
	"net/url"
	"os"
	"net/http/httputil"
	"encoding/json"
	"github.com/go-redis/redis"
)

// ReverseResponse object contains the response from reverse-proxy
type ReverseResponse struct {
	response string
	proxy *httputil.ReverseProxy
	request *http.Request
}

func serveReverseProxy(res http.ResponseWriter, req *http.Request, redisClient *redis.Client) {
	url, _ := url.Parse(os.Getenv("REVERSE_PROXY"))
	ctx := req.Context()

	responses := make(chan ReverseResponse)
	go func() {
		responses<- getRequestInCache(req.Host + req.URL.Path, redisClient)
		responses<- requestReverseProxy(req, url, redisClient)
	}()

	response := <-responses

	if http.MethodGet == req.Method && "" != response.response {
		var responseJSON RequestResponse
		err := json.Unmarshal([]byte(response.response), &responseJSON)
		if err != nil {
			panic(err)
		}
		
		for k, v := range responseJSON.Headers {
			res.Header().Set(k, v[0])
		}
		res.Write(responseJSON.Body)
	} else {
		req = req.WithContext(ctx)
		response2 := <-responses
		response2.proxy.ServeHTTP(res, req)
	}
}

// Start cache system
func Start() {
	redisClient := redisClientConnectionFactory()
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveReverseProxy(writer, request, redisClient)
	})
	go func() {
		if err := http.ListenAndServeTLS(":" + os.Getenv("CACHE_TLS_PORT"), "server.crt", "server.key", nil); err != nil {
			panic(err)
		}
	}()
	if err := http.ListenAndServe(":" + os.Getenv("CACHE_PORT"), nil); err != nil {
		panic(err)
	}

}
