package traefik

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
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
	"github.com/pquerna/cachecontrol/cacheobject"
)

type getterContext struct {
	rw   http.ResponseWriter
	req  *http.Request
	next http.Handler
}

// CustomWriter is a custom writer
type CustomWriter struct {
	Response *http.Response
	Buf      *bytes.Buffer
	Rw       http.ResponseWriter
}

// Header will write the response headers
func (r *CustomWriter) Header() http.Header {
	return r.Rw.Header()
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
	if r.Response == nil {
		r.Response = &http.Response{}
	}
	if code != 0 {
		r.Response.StatusCode = code
	}
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.Response.Header = r.Header()
	r.Buf.Write(b)
	r.Response.Body = ioutil.NopCloser(r.Buf)
	return len(b), nil
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	for h, v := range r.Response.Header {
		if len(v) > 0 {
			r.Rw.Header().Set(h, strings.Join(v, ", "))
		}
	}
	r.Rw.WriteHeader(r.Response.StatusCode)
	b, _ := ioutil.ReadAll(r.Response.Body)
	return r.Rw.Write(b)
}

func canHandle(r *http.Request, re types.RetrieverResponsePropertiesInterface) bool {
	co, err := cacheobject.ParseResponseCacheControl(r.Header.Get("Cache-Control"))
	return err == nil && len(co.NoCache) == 0 && r.Header.Get("Upgrade") != "websocket" && (re.GetExcludeRegexp() == nil || !re.GetExcludeRegexp().MatchString(r.RequestURI))
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
		Rw:       rw,
		Response: &http.Response{},
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

	if !strings.Contains(req.Header.Get("Cache-Control"), "no-cache") {
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
