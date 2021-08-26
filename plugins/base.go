package plugins

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

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
	rc coalescing.RequestCoalescingInterface,
	nextMiddleware func(w http.ResponseWriter, r *http.Request) error,
) {
	coalesceable := make(chan bool)
	responses := make(chan types.ReverseResponse)
	cacheCandidate := http.MethodGet == req.Method && !strings.Contains(req.Header.Get("Cache-Control"), "no-cache")

	go func() {
		cacheKey := rfc.GetCacheKey(req)
		go func() {
			coalesceable <- retriever.GetTransport().GetCoalescingLayerStorage().Exists(cacheKey)
		}()
		if cacheCandidate {
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

	if cacheCandidate {
		response, open := <-responses
		if open && nil != response.Response {
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
	if <-coalesceable {
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
	transport := rfc.NewTransport(provider, ykeys.InitializeYKeys(c.GetYkeys()))
	c.GetLogger().Debug("Transport initialized.")

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
	retriever.Transport.SetURL(retriever.MatchedURL)
	retriever.GetConfiguration().GetLogger().Info("Souin configuration is now loaded.")

	return retriever
}

// SouinBasePlugin declaration.
type SouinBasePlugin struct {
	Retriever         types.RetrieverResponsePropertiesInterface
	RequestCoalescing coalescing.RequestCoalescingInterface
}
