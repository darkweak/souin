package traefik

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/surrogate"
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

// CustomWriter is a custom writer
type CustomWriter struct {
	Response *http.Response
	BufPool  *sync.Pool
	http.ResponseWriter
}

// Header will write the response headers
func (r *CustomWriter) Header() http.Header {
	return r.ResponseWriter.Header()
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
	if code == 0 {
		return
	}
	r.Response.StatusCode = code
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.Response.Header = r.ResponseWriter.Header()
	if r.Response.StatusCode == 0 {
		r.Response.StatusCode = http.StatusOK
	}
	r.Response.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return len(b), nil
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	b, _ := ioutil.ReadAll(r.Response.Body)
	for h, v := range r.Response.Header {
		if len(v) > 0 {
			r.Header().Set(h, strings.Join(v, ", "))
		}
	}
	r.ResponseWriter.WriteHeader(r.Response.StatusCode)
	return r.ResponseWriter.Write(b)
}

type key string

const getterContextCtxKey key = "getter_context"

// InitializeRequest generate a CustomWriter instance to handle the response
func InitializeRequest(rw http.ResponseWriter, req *http.Request, next http.Handler) *CustomWriter {
	getterCtx := getterContext{rw, req, next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	return &CustomWriter{
		ResponseWriter: rw,
		Response:       &http.Response{},
	}
}

func sendAnyCachedResponse(rh http.Header, response *http.Response, res http.ResponseWriter) {
	response.Header = rh
	for k, v := range response.Header {
		res.Header().Set(k, v[0])
	}
	res.WriteHeader(response.StatusCode)
	b, _ := ioutil.ReadAll(response.Body)
	_, _ = res.Write(b)
	_, _ = res.(*CustomWriter).Send()
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
	retriever.SetMatchedURLFromRequest(req)

	if http.MethodGet == req.Method && !strings.Contains(req.Header.Get("Cache-Control"), "no-cache") {
		r, _ := rfc.CachedResponse(
			retriever.GetProvider(),
			req,
			cacheKey,
			retriever.GetTransport(),
			false,
		)

		if r != nil {
			rh := r.Header
			rfc.HitCache(&rh, retriever.GetMatchedURL().TTL.Duration)
			sendAnyCachedResponse(rh, r, res)
			return
		}

		r, _ = rfc.CachedResponse(
			retriever.GetProvider(),
			req,
			"STALE_"+cacheKey,
			retriever.GetTransport(),
			false,
		)

		if r != nil {
			rh := r.Header
			rfc.HitStaleCache(&rh, retriever.GetMatchedURL().TTL.Duration)
			sendAnyCachedResponse(rh, r, res)
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
	transport = rfc.NewTransport(provider, ykeys.InitializeYKeys(c.GetYkeys()), surrogate.InitializeSurrogate(c))
	var excludedRegexp *regexp.Regexp = nil
	if c.GetDefaultCache().GetRegex().Exclude != "" {
		excludedRegexp = regexp.MustCompile(c.GetDefaultCache().GetRegex().Exclude)
	}

	retriever := &types.RetrieverResponseProperties{
		MatchedURL: configurationtypes.URL{
			TTL:     configurationtypes.Duration{Duration: c.GetDefaultCache().GetTTL()},
			Headers: c.GetDefaultCache().GetHeaders(),
		},
		Provider:      provider,
		Configuration: c,
		RegexpUrls:    regexpUrls,
		Transport:     transport,
		ExcludeRegex:  excludedRegexp,
	}

	retriever.Transport.SetURL(retriever.MatchedURL)
	return retriever
}

// SouinBasePlugin declaration.
type SouinBasePlugin struct {
	Retriever         types.RetrieverResponsePropertiesInterface
	RequestCoalescing coalescing.RequestCoalescingInterface
	MapHandler        *api.MapHandler
}

// HandleInternally handles the Souin custom endpoints
func (s *SouinBasePlugin) HandleInternally(r *http.Request) (bool, http.HandlerFunc) {
	if s.MapHandler != nil {
		for k, souinHandler := range *s.MapHandler.Handlers {
			if strings.Contains(r.RequestURI, k) {
				return true, souinHandler
			}
		}
	}

	return false, nil
}
