package traefik

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
)

type getterContext struct {
	rw   http.ResponseWriter
	req  *http.Request
	next http.Handler
}

type customWriter struct {
	Response *http.Response
	http.ResponseWriter
}

func (r *customWriter) Write(b []byte) (int, error) {
	r.Response.Header = r.ResponseWriter.Header()
	r.Response.StatusCode = http.StatusOK
	r.Response.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return r.ResponseWriter.Write(b)
}

type key string

const getterContextCtxKey key = "getter_context"

func InitializeRequest(rw http.ResponseWriter, req *http.Request, next http.Handler) *customWriter {
	getterCtx := getterContext{rw, req, next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	return &customWriter{
		ResponseWriter: rw,
		Response:       &http.Response{},
	}
}

// DefaultSouinPluginCallback is the default callback for plugins
func DefaultSouinPluginCallback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	_ coalescing.RequestCoalescingInterface,
	nextMiddleware func(w http.ResponseWriter, r *http.Request) error,
) {
	cacheKey := rfc.GetCacheKey(req)
	if http.MethodGet == req.Method {
		r, _ := rfc.CachedResponse(
			retriever.GetProvider(),
			req,
			cacheKey,
			retriever.GetTransport(),
			true,
		)

		m := r.Response
		if !(m == nil) {
			rh := r.Response.Header
			rfc.HitCache(&rh)
			r.Response.Header = rh
			for k, v := range r.Response.Header {
				res.Header().Set(k, v[0])
			}
			res.WriteHeader(r.Response.StatusCode)
			b, _ := ioutil.ReadAll(r.Response.Body)
			_, _ = res.Write(b)
			return
		}
	}

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
