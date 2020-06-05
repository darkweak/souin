package cache

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	cacheProviders "github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/providers"
	"github.com/darkweak/souin/configuration"
)

func serveReverseProxy(
	res http.ResponseWriter,
	req *http.Request,
	providers *[]cacheProviders.AbstractProviderInterface,
	configurationInstance configuration.Configuration,
) {
	url, _ := url.Parse(configurationInstance.ReverseProxyURL)
	ctx := req.Context()

	responses := make(chan types.ReverseResponse)
	go func() {
		for _, v := range *providers {
			responses <- v.GetRequestInCache(string(req.Host + req.URL.Path))
		}
		responses <- service.RequestReverseProxy(req, url, *providers, configurationInstance)
	}()
	alreadySent := false

	for i := 0; i < len(*providers); i++ {
		response := <-responses
		if http.MethodGet == req.Method && "" != response.Response {
			var responseJSON types.RequestResponse
			err := json.Unmarshal([]byte(response.Response), &responseJSON)
			if err != nil {
				panic(err)
			}
			for k, v := range responseJSON.Headers {
				res.Header().Set(k, v[0])
			}
			alreadySent = true
			res.Write(responseJSON.Body)
		}
	}

	if !alreadySent {
		req = req.WithContext(ctx)
		response2 := <-responses
		response2.Proxy.ServeHTTP(res, req)
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
		err := server.Serve(listener)
		if nil != err {
			fmt.Println(err)
		}
	}()

	return listener, &server
}

// Start cache system
func Start() {
	configurationInstance := *configuration.GetConfig()
	providersList := cacheProviders.InitializeProviders(configurationInstance)

	configChannel := make(chan int)
	tlsconfig := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		NameToCertificate:  make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	tlsconfig.Certificates = append(tlsconfig.Certificates, v)

	go func() {
		providers.InitProviders(tlsconfig, &configChannel)
	}()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveReverseProxy(writer, request, providersList, configurationInstance)
	})
	go func() {
		listener, _ := startServer(tlsconfig)
		for {
			select {
			case <-configChannel:
				listener.Close()
				listener, _ = startServer(tlsconfig)
			}
		}

	}()
	if err := http.ListenAndServe(":"+configurationInstance.Cache.Port.Web, nil); err != nil {
		panic(err)
	}

}
