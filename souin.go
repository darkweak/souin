package main

import (
	"net/http"
	"net/url"
	"os"
	"net/http/httputil"
	"encoding/json"
)

type ReverseResponse struct {
	response string
	proxy *httputil.ReverseProxy
	request *http.Request
}

func serveReverseProxy(res http.ResponseWriter, req *http.Request) {
	url, _ := url.Parse(os.Getenv("REVERSE_PROXY"))

	responses := make(chan ReverseResponse)
	go func() {
		responses<- getRequestInCache(req.Host + req.URL.Path)
		responses<- requestReverseProxy(req, url)
	}()

	response := <-responses

	if "" != response.response {
		var responseJson RequestResponse
		json.Unmarshal([]byte(response.response), &responseJson)
		for k, v := range responseJson.Headers {
			res.Header().Set(k, v[0])
		}
		res.Write(responseJson.Body)
	} else {
		response2 := <-responses
		response2.proxy.ServeHTTP(res, req)
	}
}

func main() {
	http.HandleFunc("/", serveReverseProxy)
	if err := http.ListenAndServe(":" + os.Getenv("CACHE_PORT"), nil); err != nil {
		panic(err)
	}
}
