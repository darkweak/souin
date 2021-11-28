package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/plugins/souin/configuration"
	"github.com/darkweak/souin/plugins/souin/providers"
	souintypes "github.com/darkweak/souin/plugins/souin/types"
)

func souinPluginInitializerFromConfiguration(c *configuration.Configuration) *souintypes.SouinRetrieverResponseProperties {
	baseRetriever := *plugins.DefaultSouinPluginInitializerFromConfiguration(c)
	u, err := url.Parse(c.GetReverseProxyURL())
	if err != nil {
		panic("Reverse proxy url is missing")
	}

	retriever := &souintypes.SouinRetrieverResponseProperties{
		RetrieverResponseProperties: baseRetriever,
		ReverseProxyURL:             u,
	}

	return retriever
}

func startServer(config *tls.Config, port string) (net.Listener, *http.Server) {
	server := http.Server{
		Addr:      ":"+port,
		Handler:   nil,
		TLSConfig: config,
	}
	listener, err := tls.Listen("tcp", ":"+port, config)
	if err != nil {
		fmt.Println(err)
	}
	go func() {
		if e := server.Serve(listener); nil != e {
			fmt.Println(e)
		}
	}()

	return listener, &server
}

func main() {
	c := configuration.GetConfiguration()
	rc := coalescing.Initialize()
	configChannel := make(chan int)
	tlsConfig := &tls.Config{
		Certificates: make([]tls.Certificate, 0),
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
	for _, endpoint := range api.Initialize(retriever.GetTransport(), c) {
		if endpoint.IsEnabled() {
			c.GetLogger().Info(fmt.Sprintf("Enabling %s%s endpoint.", basePathAPIS, endpoint.GetBasePath()))
			http.HandleFunc(fmt.Sprintf("%s%s", basePathAPIS, endpoint.GetBasePath()), endpoint.HandleRequest)
			http.HandleFunc(fmt.Sprintf("%s%s/", basePathAPIS, endpoint.GetBasePath()), endpoint.HandleRequest)
		}
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		request.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
		retriever.SetMatchedURLFromRequest(request)
		coalescing.ServeResponse(writer, request, retriever, plugins.DefaultSouinPluginCallback, rc, func(w http.ResponseWriter, r *http.Request) error {
			rr := service.RequestReverseProxy(r, *retriever)
			select {
			case <-r.Context().Done():
				c.GetLogger().Debug("The request was canceled by the user.")
				return &errors.CanceledRequestContextError{}
			default:
				rr.Proxy.ServeHTTP(w, r)
			}
			return nil
		})
	})
	go func() {
		for {
			listener, _ := startServer(tlsConfig, c.DefaultCache.Port.TLS)
			<-configChannel
			_ = listener.Close()
		}
	}()

	c.GetLogger().Debug("Waiting for an incoming request...")
	if err := http.ListenAndServe(":"+c.DefaultCache.Port.Web, nil); err != nil {
		panic(err)
	}
}
