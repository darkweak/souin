package main

import (
	"crypto/tls"
	"fmt"
	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configuration"
	"github.com/darkweak/souin/plugins"
	souintypes "github.com/darkweak/souin/plugins/souin/types"
	"github.com/darkweak/souin/providers"
	"github.com/darkweak/souin/rfc"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

func callback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	rc coalescing.RequestCoalescingInterface,
	nm func(w http.ResponseWriter, r *http.Request) error,
) {
	responses := make(chan types.ReverseResponse)

	go func() {
		if http.MethodGet == req.Method {
			r, _ := rfc.CachedResponse(
				retriever.GetProvider(),
				req,
				rfc.GetCacheKey(req),
				retriever.GetTransport(),
				true,
			)
			responses <- r
			if nil != r.Response {
				return
			}
		}
	}()

	if http.MethodGet == req.Method {
		response, open := <-responses
		if open && nil != response.Response {
			close(responses)
			for k, v := range response.Response.Header {
				res.Header().Set(k, v[0])
			}
			b, _ := ioutil.ReadAll(response.Response.Body)
			_, _ = res.Write(b)
			return
		}
	}

	close(responses)
	rc.Temporise(req, res, nm)
}

func souinPluginInitializerFromConfiguration(c *configuration.Configuration) *souintypes.SouinRetrieverResponseProperties {
	baseRetriever := *plugins.DefaultSouinPluginInitializerFromConfiguration(c)
	u, err := url.Parse(c.GetReverseProxyURL())
	if err != nil {
		panic("Reverse proxy url is missing")
	}

	retriever := &souintypes.SouinRetrieverResponseProperties{
		RetrieverResponseProperties: baseRetriever,
		ReverseProxyURL: u,
	}

	return retriever
}

func startServer(config *tls.Config) (net.Listener, *http.Server) {
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

func main() {
	c := configuration.GetConfiguration()
	rc := coalescing.Initialize()
	configChannel := make(chan int)
	tlsConfig := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("./default/server.crt", "./default/server.key")
	tlsConfig.Certificates = append(tlsConfig.Certificates, v)

	go func() {
		providers.InitProviders(tlsConfig, &configChannel, c)
	}()

	retriever := souinPluginInitializerFromConfiguration(c)

	basePathAPIS := c.GetAPI().BasePath
	if basePathAPIS == "" {
		basePathAPIS = "/souin-api"
	}
	for _, endpoint := range api.Initialize(retriever.Provider, c) {
		if endpoint.IsEnabled() {
			http.HandleFunc(fmt.Sprintf("%s%s", basePathAPIS, endpoint.GetBasePath()), endpoint.HandleRequest)
			http.HandleFunc(fmt.Sprintf("%s%s/", basePathAPIS, endpoint.GetBasePath()), endpoint.HandleRequest)
		}
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		request.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		coalescing.ServeResponse(writer, request, retriever, callback, rc, func(w http.ResponseWriter, r *http.Request) error {
			rr := service.RequestReverseProxy(r, *retriever)
			rr.Proxy.ServeHTTP(w, r)
			return nil
		})
	})
	go func() {
		listener, _ := startServer(tlsConfig)
		for {
			select {
			case <-configChannel:
				_ = listener.Close()
				listener, _ = startServer(tlsConfig)
			}
		}
	}()
	if err := http.ListenAndServe(":"+c.GetDefaultCache().Port.Web, nil); err != nil {
		panic(err)
	}
}
