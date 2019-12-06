package cache

import (
	"net/http"
	"net/url"
	"os"
	"net/http/httputil"
	"encoding/json"
	"github.com/go-redis/redis"
	"crypto/tls"
	"log"
	"github.com/darkweak/souin/providers"
	"fmt"
	"time"
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
		responses <- getRequestInCache(req.Host+req.URL.Path, redisClient)
		responses <- requestReverseProxy(req, url, redisClient)
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
	tlsconfig := &tls.Config{
		Certificates: make([]tls.Certificate, 0),
		NameToCertificate: make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	tlsconfig.Certificates = append(tlsconfig.Certificates, v)
	certificates := providers.CommonProvider{
		Certificates: make(map[string]providers.Certificate),
	}

	go func() {
		providers.InitProviders(&certificates, tlsconfig)
	}()
	time.Sleep(10 * time.Second)
	tlsconfig.BuildNameToCertificate()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveReverseProxy(writer, request, redisClient)
	})
	go func() {
		server := http.Server{
			Addr:      ":443",
			Handler:   nil,
			TLSConfig: tlsconfig,
		}
		listener, err := tls.Listen("tcp", ":443", tlsconfig)
		if err != nil {
			fmt.Println(err)
		}
		log.Fatal(server.Serve(listener))
	}()
	if err := http.ListenAndServe(":"+os.Getenv("CACHE_PORT"), nil); err != nil {
		panic(err)
	}

}
