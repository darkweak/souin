package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"net/url"
	"strings"
	"time"

	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins/souin/configuration"
	"github.com/darkweak/souin/plugins/souin/providers"
)

type canceledRequestContextError struct{}

func (c *canceledRequestContextError) Error() string {
	return "The user canceled the request"
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

func main() {
	c := configuration.GetConfiguration()
	fmt.Printf("%+v\n\n", c)
	configChannel := make(chan int)
	tlsConfig := &tls.Config{
		Certificates: make([]tls.Certificate, 0),
	}
	v, _ := tls.LoadX509KeyPair("./default/server.crt", "./default/server.key")
	tlsConfig.Certificates = append(tlsConfig.Certificates, v)

	go func() {
		providers.InitProviders(tlsConfig, &configChannel, c)
	}()

	httpCache := middleware.NewHTTPCacheHandler(c)
	reverseProxyURL, err := url.Parse(c.GetReverseProxyURL())
	if err != nil {
		panic("Invalid reverse proxy url, " + c.GetReverseProxyURL() + " given with resulting error " + err.Error())
	}

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		_ = httpCache.ServeHTTP(writer, request, func(w http.ResponseWriter, req *http.Request) error {
			req.URL.Host = req.Host
			req.URL.Scheme = reverseProxyURL.Scheme
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

			proxy := httputil.NewSingleHostReverseProxy(reverseProxyURL)
			proxy.Transport = &http.Transport{
				Proxy:               http.ProxyURL(reverseProxyURL),
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
				TLSHandshakeTimeout: 10 * time.Second,
			}

			select {
			case <-req.Context().Done():
				c.GetLogger().Debug("The request was canceled by the user.")
				return &canceledRequestContextError{}
			default:
				res, err := proxy.Transport.RoundTrip(req)
				if err != nil {
					return err
				}
				for h, hv := range res.Header {
					w.Header().Set(h, strings.Join(hv, ", "))
				}
				w.WriteHeader(res.StatusCode)

				body, _ := io.ReadAll(res.Body)
				defer res.Body.Close()
				res.Body = io.NopCloser(bytes.NewBuffer(body))
				_, err = w.Write(body)

				return err
			}
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
