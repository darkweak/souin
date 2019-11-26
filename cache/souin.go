package cache

import (
	"net/http"
	"net/url"
	"os"
	"net/http/httputil"
	"encoding/json"
	"github.com/go-redis/redis"
	"crypto/tls"
	"net"
	"fmt"
)

// ReverseResponse object contains the response from reverse-proxy
type ReverseResponse struct {
	response string
	proxy    *httputil.ReverseProxy
	request  *http.Request
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

func startServer(tlsconfig *tls.Config) (net.Listener, *http.Server) {
	tlsconfig.BuildNameToCertificate()
	server := http.Server{
		Addr:      ":443",
		Handler:   nil,
		TLSConfig: tlsconfig,
	}
	listener, err := tls.Listen("tcp", ":443", tlsconfig)
	if err != nil {
		fmt.Println(err)
	}
	go func() {
		error := server.Serve(listener)
		fmt.Println("YO")
		fmt.Println(error)
		fmt.Println("LO")
		if nil != error {
			fmt.Println(error)
		}
	}()

	return listener, &server
}

// Start cache system
func Start() {
	redisClient := redisClientConnectionFactory()
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveReverseProxy(writer, request, redisClient)
	})
	go func() {
		providers.InitProviders(&certificates, tlsconfig, &configChannel)
	}()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveReverseProxy(writer, request, redisClient)
	})
	go func() {
		listener, server := startServer(tlsconfig)
		for {
			select {
			case <- configChannel:
				listener.Close()
				if err := server.Shutdown(context.Background()); err != nil {
					fmt.Errorf("Shutdown failed: %s", err)
				}
				listener, server = startServer(tlsconfig)
			}
		}

	}()
	if err := http.ListenAndServe(":"+os.Getenv("CACHE_PORT"), nil); err != nil {
		panic(err)
	}

}
