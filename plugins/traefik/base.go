package traefik

import (
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"io/ioutil"
	"net/http"
)

// DefaultSouinPluginCallback is the default callback for plugins
func DefaultSouinPluginCallback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	_ coalescing.RequestCoalescingInterface,
	nextMiddleware func(w http.ResponseWriter, r *http.Request) error,
) {
	responses := make(chan types.ReverseResponse)

	go func() {
		cacheKey := rfc.GetCacheKey(req)
		if http.MethodGet == req.Method {
			r, _ := rfc.CachedResponse(
				retriever.GetProvider(),
				req,
				cacheKey,
				retriever.GetTransport(),
				true,
			)
			responses <- r
		}
	}()

	if http.MethodGet == req.Method {
		response := <-responses
		m := response.Response
		if !(m == nil) {
			close(responses)
			rh := response.Response.Header
			rfc.HitCache(&rh)
			response.Response.Header = rh
			for k, v := range response.Response.Header {
				res.Header().Set(k, v[0])
			}
			res.WriteHeader(response.Response.StatusCode)
			b, _ := ioutil.ReadAll(response.Response.Body)
			_, _ = res.Write(b)
			return
		}
	}

	close(responses)
	_ = nextMiddleware(res, req)
}

// DefaultSouinPluginInitializerFromConfiguration is the default initialization for plugins
func DefaultSouinPluginInitializerFromConfiguration(c configurationtypes.AbstractConfigurationInterface) *types.RetrieverResponseProperties {
	provider := providers.InitializeProvider(c)
	regexpUrls := helpers.InitializeRegexp(c)
	var transport types.TransportInterface
	transport = rfc.NewTransport(provider, ykeys.InitializeYKeys(c.GetYkeys()))

	retriever := &types.RetrieverResponseProperties{
		MatchedURL: configurationtypes.URL{
			TTL:     configurationtypes.Duration{Duration: c.GetDefaultCache().GetTTL()},
			Headers: c.GetDefaultCache().GetHeaders(),
		},
		Provider:      provider,
		Configuration: c,
		RegexpUrls:    regexpUrls,
		Transport:     transport,
	}
	return retriever
}

// SouinBasePlugin declaration.
type SouinBasePlugin struct {
	Retriever         types.RetrieverResponsePropertiesInterface
	RequestCoalescing coalescing.RequestCoalescingInterface
}
