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
	"strings"
	"regexp"
)

// InitializeRegexp will generate one strong regex from your urls defined in the configuration.yml
func InitializeRegexp(configurationInstance configuration.Configuration) regexp.Regexp {
	u := ""
	for k := range configurationInstance.URLs {
		if "" != u {
			u += "|"
		}
		u += "(" + k + ")"
	}

	return *regexp.MustCompile(u)
}

func serveReverseProxy(
	res http.ResponseWriter,
	req *http.Request,
	providers map[string]cacheProviders.AbstractProviderInterface,
	configurationInstance configuration.Configuration,
	regexpUrls regexp.Regexp,
	matchedURL configuration.URL,
) {
	path := req.Host + req.URL.Path

	regexpURL := regexpUrls.FindString(path)
	if "" != regexpURL {
		matchedURL = configurationInstance.URLs[regexpURL]
	}

	u, _ := url.Parse(configurationInstance.ReverseProxyURL)
	ctx := req.Context()

	responses := make(chan types.ReverseResponse)
	headers := ""
	if matchedURL.Headers != nil && len(matchedURL.Headers) > 0 {
		for _, h := range matchedURL.Headers {
			headers += strings.ReplaceAll(req.Header.Get(h), " ", "")
		}
	}

	alreadyHaveResponse := false
	alreadySent := false

	go func() {
		if http.MethodGet == req.Method {
			for _, v := range matchedURL.Providers {
				if !alreadyHaveResponse {
					pr := providers[v]
					p := string(path + headers)
					r := pr.GetRequestInCache(p)
					responses <- pr.GetRequestInCache(p)
					if "" != r.Response {
						alreadyHaveResponse = true
					}
				}
			}
		}
		if !alreadyHaveResponse || http.MethodGet != req.Method {
			responses <- service.RequestReverseProxy(req, u, providers, configurationInstance, matchedURL)
		}
	}()

	if http.MethodGet == req.Method {
		for range matchedURL.Providers {
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
	}

	if !alreadySent {
		req = req.WithContext(ctx)
		response2 := <-responses
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
	configurationInstance := configuration.GetConfig()
	providersList := cacheProviders.InitializeProviders(configurationInstance)
	regexpUrls := InitializeRegexp(configurationInstance)

	configChannel := make(chan int)
	tlsconfig := &tls.Config{
		Certificates:       make([]tls.Certificate, 0),
		NameToCertificate:  make(map[string]*tls.Certificate),
		InsecureSkipVerify: true,
	}
	v, _ := tls.LoadX509KeyPair("server.crt", "server.key")
	tlsconfig.Certificates = append(tlsconfig.Certificates, v)

	go func() {
		providers.InitProviders(tlsconfig, &configChannel, configurationInstance)
	}()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		serveReverseProxy(
			writer,
			request,
			providersList,
			configurationInstance,
			regexpUrls,
			configuration.URL{
				TTL:       configurationInstance.DefaultCache.TTL,
				Providers: configurationInstance.DefaultCache.Providers,
				Headers:   configurationInstance.DefaultCache.Headers,
			},
		)
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
	if err := http.ListenAndServe(":"+configurationInstance.DefaultCache.Port.Web, nil); err != nil {
		panic(err)
	}

}
