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
)

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
		service.ServeResponse(writer, request, retriever)
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
