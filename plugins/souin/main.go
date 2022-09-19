package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/service"
	"github.com/darkweak/souin/errors"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/plugins/souin/configuration"
	"github.com/darkweak/souin/plugins/souin/providers"
	souintypes "github.com/darkweak/souin/plugins/souin/types"
	"github.com/darkweak/souin/rfc"
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
		Addr:      ":" + port,
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

var bufPool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
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
		request = retriever.GetContext().SetBaseContext(request)

		if !plugins.CanHandle(request, retriever) {
			rfc.MissCache(writer.Header().Set, request)
			return
		}

		request = retriever.GetContext().SetContext(request)
		callback := func(rw http.ResponseWriter, rq *http.Request, ret souintypes.SouinRetrieverResponseProperties) error {
			rr := service.RequestReverseProxy(rq, ret)
			select {
			case <-rq.Context().Done():
				c.GetLogger().Debug("The request was canceled by the user.")
				return &errors.CanceledRequestContextError{}
			default:
				res, _ := rr.Proxy.Transport.RoundTrip(rq)
				for h, hv := range res.Header {
					rw.Header().Set(h, strings.Join(hv, ", "))
				}
				rw.WriteHeader(res.StatusCode)

				body, _ := io.ReadAll(res.Body)
				defer res.Body.Close()
				res.Body = io.NopCloser(bytes.NewBuffer(body))
				_, _ = rw.Write(body)
			}

			return nil
		}
		if plugins.HasMutation(request, writer) {
			_ = callback(writer, request, *retriever)
			return
		}
		customWriter := &plugins.CustomWriter{
			Response: &http.Response{},
			Buf:      bufPool.Get().(*bytes.Buffer),
			Rw:       writer,
			Req:      request,
		}
		retriever.SetMatchedURLFromRequest(request)
		_ = plugins.DefaultSouinPluginCallback(customWriter, request, retriever, rc, func(w http.ResponseWriter, r *http.Request) (e error) {
			_ = callback(customWriter, request, *retriever)
			request.Response = customWriter.Response
			request.Response, e = retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(request)
			return
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
