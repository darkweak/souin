package plugins

import (
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"net/http"
)

// DefaultSouinPluginCallback is the default callback for plugins
func DefaultSouinPluginCallback(
	res http.ResponseWriter,
	req *http.Request,
	retriever types.RetrieverResponsePropertiesInterface,
	rc coalescing.RequestCoalescingInterface,
	nextMiddleware func(w http.ResponseWriter, r *http.Request) error,
) {
	responses := make(chan types.ReverseResponse)
	coalesceable := make(chan bool)

	go func() {
		cacheKey := rfc.GetCacheKey(req)
		varied := retriever.GetTransport().GetVaryLayerStorage().Get(cacheKey)
		if len(varied) != 0 {
			cacheKey = rfc.GetVariedCacheKey(req, varied)
		}
		go func() {
			coalesceable <- retriever.GetTransport().GetCoalescingLayerStorage().Exists(cacheKey)
		}()
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
		response, open := <-responses
		if open && nil != response.Response {
			close(responses)
			rh := response.Response.Header
			rfc.HitCache(&rh)
			response.Response.Header = rh
			for k, v := range response.Response.Header {
				res.Header().Set(k, v[0])
			}
			b, _ := ioutil.ReadAll(response.Response.Body)
			_, _ = res.Write(b)
			return
		}
	}

	close(responses)
	if <-coalesceable {
		rc.Temporise(req, res, nextMiddleware)
	} else {
		_ = nextMiddleware(res, req)
	}
	close(coalesceable)
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
	c.GetLogger().Debug("Provider initialized")
	regexpUrls := helpers.InitializeRegexp(c)
	var transport types.TransportInterface
	transport = rfc.NewTransport(provider)
	c.GetLogger().Debug("Transport initialized")

	retriever := &types.RetrieverResponseProperties{
		MatchedURL: configurationtypes.URL{
			TTL:     c.GetDefaultCache().GetTTL(),
			Headers: c.GetDefaultCache().GetHeaders(),
		},
		Provider:      provider,
		Configuration: c,
		RegexpUrls:    regexpUrls,
		Transport:     transport,
	}
	retriever.Transport.SetURL(retriever.MatchedURL)
	retriever.GetConfiguration().GetLogger().Debug("Souin configuration is now loaded")

	return retriever
}

// SouinBasePlugin declaration.
type SouinBasePlugin struct {
	Retriever         types.RetrieverResponsePropertiesInterface
	RequestCoalescing coalescing.RequestCoalescingInterface
}
