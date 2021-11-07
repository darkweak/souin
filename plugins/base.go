package plugins

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// CustomWriter handles the response and provide the way to cache the value
type CustomWriter struct {
	Response *http.Response
	Buf      *bytes.Buffer
	Rw       http.ResponseWriter
}

// WriteHeader will write the response headers
func (r *CustomWriter) Header() http.Header {
	return r.Rw.Header()
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
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
	b, _ := ioutil.ReadAll(r.Response.Body)
	return r.Rw.Write(b)
}

// CanHandle detect if the request can be handled by Souin
func CanHandle(r *http.Request, re types.RetrieverResponsePropertiesInterface) bool {
	return r.Header.Get("Upgrade") != "websocket" && (re.GetExcludeRegexp() == nil || !re.GetExcludeRegexp().MatchString(r.RequestURI))
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
) {
	coalesceable := make(chan bool)
	responses := make(chan *http.Response)
	cacheCandidate := http.MethodGet == req.Method && !strings.Contains(req.Header.Get("Cache-Control"), "no-cache")
	cacheKey := rfc.GetCacheKey(req)

	go func() {
		coalesceable <- retriever.GetTransport().GetCoalescingLayerStorage().Exists(cacheKey)
	}()

	if cacheCandidate {
		go func() {
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
		for i := 0; i < 2; i++ {
			response, open := <-responses
			if open && nil != response {
				rh := response.Header
				rfc.HitCache(&rh)
				sendAnyCachedResponse(rh, response, res)
				return
			}
		}
	}

	close(responses)
	if <-coalesceable && rc != nil {
		rc.Temporize(req, res, nextMiddleware)
	} else {
		_ = nextMiddleware(res, req)
	}
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
