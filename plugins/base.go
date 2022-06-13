package plugins

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/api/prometheus"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"github.com/pquerna/cachecontrol/cacheobject"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CustomWriter handles the response and provide the way to cache the value
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
	r.Response.Header = r.Rw.Header()
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
	var b []byte

	if r.Response.Body != nil {
		b, _ = ioutil.ReadAll(r.Response.Body)
	}
	return r.Rw.Write(b)
}

func HasMutation(req *http.Request, rw http.ResponseWriter) bool {
	if req.Context().Value(context.IsMutationRequest).(bool) {
		rw.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
		return true
	}
	return false
}

// CanHandle detect if the request can be handled by Souin
func CanHandle(r *http.Request, re types.RetrieverResponsePropertiesInterface) bool {
	co, err := cacheobject.ParseResponseCacheControl(r.Header.Get("Cache-Control"))
	return r.Context().Value(context.SupportedMethod).(bool) && err == nil && !co.NoCachePresent && r.Header.Get("Upgrade") != "websocket" && (re.GetExcludeRegexp() == nil || !re.GetExcludeRegexp().MatchString(r.RequestURI))
}

func sendAnyCachedResponse(rh http.Header, response *http.Response, res http.ResponseWriter) {
	response.Header = rh
	for k, v := range response.Header {
		res.Header().Set(k, v[0])
	}
	res.WriteHeader(response.StatusCode)
	b, _ := ioutil.ReadAll(response.Body)
	_, _ = res.Write(b)
	cw, success := res.(*CustomWriter)
	if success {
		_, _ = cw.Send()
	}
}

// DefaultSouinPluginCallback is the default callback for plugins
func DefaultSouinPluginCallback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	rc coalescing.RequestCoalescingInterface,
	nextMiddleware func(w http.ResponseWriter, r *http.Request) error,
) (e error) {
	prometheus.Increment(prometheus.RequestCounter)
	start := time.Now()
	coalesceable := make(chan bool)
	responses := make(chan *http.Response)
	defer func() {
		close(coalesceable)
		close(responses)
	}()
	cacheCandidate := !strings.Contains(req.Header.Get("Cache-Control"), "no-cache")
	cacheKey := req.Context().Value(context.Key).(string)
	retriever.SetMatchedURLFromRequest(req)

	go func() {
		defer func() {
			_ = recover()
		}()
		coalesceable <- retriever.GetTransport().GetCoalescingLayerStorage().Exists(cacheKey)
	}()

	if cacheCandidate {
		go func() {
			defer func() {
				_ = recover()
			}()
			r, _ := rfc.CachedResponse(
				retriever.GetProvider(),
				req,
				cacheKey,
				retriever.GetTransport(),
				false,
			)

			responses <- rfc.ValidateMaxAgeCachedResponse(req, r)

			r, _ = rfc.CachedResponse(
				retriever.GetProvider(),
				req,
				"STALE_"+cacheKey,
				retriever.GetTransport(),
				false,
			)

			responses <- rfc.ValidateStaleCachedResponse(req, r)
		}()
	}

	if cacheCandidate {
		response, open := <-responses
		if open && nil != response {
			rh := response.Header
			rfc.HitCache(&rh, retriever.GetMatchedURL().TTL.Duration)
			prometheus.Increment(prometheus.CachedResponseCounter)
			sendAnyCachedResponse(rh, response, res)
			return
		}

		response, open = <-responses
		if open && nil != response {
			rh := response.Header
			rfc.HitStaleCache(&rh, retriever.GetMatchedURL().TTL.Duration)
			sendAnyCachedResponse(rh, response, res)
			return
		}
	}

	prometheus.Increment(prometheus.NoCachedResponseCounter)
	if <-coalesceable && rc != nil {
		rc.Temporize(req, res, nextMiddleware)
	} else {
		e = nextMiddleware(res, req)
	}
	prometheus.Add(prometheus.AvgResponseTime, float64(time.Since(start).Milliseconds()))

	return e
}

// DefaultSouinPluginInitializerFromConfiguration is the default initialization for plugins
func DefaultSouinPluginInitializerFromConfiguration(c configurationtypes.AbstractConfigurationInterface) *types.RetrieverResponseProperties {
	var logLevel zapcore.Level
	if c.GetLogLevel() == "" {
		logLevel = zapcore.FatalLevel
	} else if err := logLevel.UnmarshalText([]byte(c.GetLogLevel())); err != nil {
		logLevel = zapcore.FatalLevel
	}
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(logLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	logger, _ := cfg.Build()
	c.SetLogger(logger)

	provider := providers.InitializeProvider(c)
	c.GetLogger().Debug("Provider initialized.")
	regexpUrls := helpers.InitializeRegexp(c)
	transport := rfc.NewTransport(provider, ykeys.InitializeYKeys(c.GetYkeys()), surrogate.InitializeSurrogate(c))
	c.GetLogger().Debug("Transport initialized.")
	var excludedRegexp *regexp.Regexp = nil
	if c.GetDefaultCache().GetRegex().Exclude != "" {
		excludedRegexp = regexp.MustCompile(c.GetDefaultCache().GetRegex().Exclude)
	}

	ctx := context.GetContext()
	ctx.Init(c)

	retriever := &types.RetrieverResponseProperties{
		MatchedURL: configurationtypes.URL{
			TTL:                 configurationtypes.Duration{Duration: c.GetDefaultCache().GetTTL()},
			Headers:             c.GetDefaultCache().GetHeaders(),
			DefaultCacheControl: c.GetDefaultCache().GetDefaultCacheControl(),
		},
		Provider:      provider,
		Configuration: c,
		RegexpUrls:    regexpUrls,
		Transport:     transport,
		ExcludeRegex:  excludedRegexp,
		Context:       ctx,
	}
	retriever.Transport.SetURL(retriever.MatchedURL)
	retriever.GetConfiguration().GetLogger().Info("Souin configuration is now loaded.")

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
