package cache

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"github.com/darkweak/souin/providers"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/cache/types"
	providers2 "github.com/darkweak/souin/cache/providers"
	"net/url"
	"encoding/json"
)

func mycallback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	key string,
) {

	u, _ := url.Parse(retriever.GetConfiguration().GetReverseProxyURL())
	ctx := req.Context()
	responses := make(chan types.ReverseResponse)

	alreadyHaveResponse := false
	alreadySent := false

	go func() {
		if http.MethodGet == req.Method {
			if !alreadyHaveResponse {
				r := retriever.GetProvider().GetRequestInCache(key)
				responses <- retriever.GetProvider().GetRequestInCache(key)
				if "" != r.Response {
					alreadyHaveResponse = true
				}
			}
		}
		if !alreadyHaveResponse || http.MethodGet != req.Method {
			responses <- service.RequestReverseProxy(req, u, retriever)
		}
	}()

	if http.MethodGet == req.Method {
		response, open := <-responses
		if open && http.MethodGet == req.Method && "" != response.Response {
			close(responses)
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
		close(responses)
		response2.Proxy.ServeHTTP(res, req)
	}
}

func startServer(config *tls.Config) (net.Listener, *http.Server) {
	config.BuildNameToCertificate()
	server := http.Server{
		Addr:      ":443",
		Handler:   nil,
		TLSConfig: config,
	}
	listener, err := tls.Listen("tcp", ":443", config)
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
	config := configuration.GetConfiguration()
	configChannel := make(chan int)
	tlsConfig := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		NameToCertificate:  make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("./default/server.crt", "./default/server.key")
	tlsConfig.Certificates = append(tlsConfig.Certificates, v)

	go func() {
		providers.InitProviders(tlsConfig, &configChannel, config)
	}()


	c := configuration.GetConfiguration()
	provider := providers2.InitializeProvider(c)
	regexpUrls := helpers.InitializeRegexp(c)
	retriever := &types.RetrieverResponseProperties{
		MatchedURL: configuration.URL{
			TTL:       c.GetDefaultCache().TTL,
			Headers:   c.GetDefaultCache().Headers,
		},
		Provider: provider,
		Configuration: c,
		RegexpUrls: regexpUrls,
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		service.ServeResponse(writer, request, retriever, mycallback)
	})
	go func() {
		listener, _ := startServer(tlsConfig)
		for {
			select {
			case <-configChannel:
				listener.Close()
				listener, _ = startServer(tlsConfig)
			}
		}
	}()
	if err := http.ListenAndServe(":"+config.GetDefaultCache().Port.Web, nil); err != nil {
		panic(err)
	}
}
