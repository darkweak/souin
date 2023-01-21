package plugins

import (
	"bytes"
	ctx "context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/darkweak/go-esi/esi"
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

var (
	serverTimeoutMessage                      = []byte("Internal server error")
	_                    souinWriterInterface = (*CustomWriter)(nil)
)

type souinWriterInterface interface {
	http.ResponseWriter
	Send() (int, error)
}

// CustomWriter handles the response and provide the way to cache the value
type CustomWriter struct {
	Response    *http.Response
	Buf         *bytes.Buffer
	Rw          http.ResponseWriter
	Req         *http.Request
	size        int
	headersSent bool
}

// Header will write the response headers
func (r *CustomWriter) Header() http.Header {
	if r.headersSent {
		return http.Header{}
	}
	return r.Rw.Header()
}

// WriteHeader will write the response headers
func (r *CustomWriter) WriteHeader(code int) {
	if r.headersSent {
		return
	}
	if r.Response == nil {
		r.Response = &http.Response{}
	}
	if code != 0 {
		r.Response.StatusCode = code
	}
	r.Response.Header = r.Rw.Header()
	r.headersSent = true
}

// Write will write the response body
func (r *CustomWriter) Write(b []byte) (int, error) {
	r.Buf.Grow(len(b))
	_, _ = r.Buf.Write(b)
	r.Response.Body = io.NopCloser(bytes.NewBuffer(r.Buf.Bytes()))
	r.size += len(b)
	r.Response.Header.Set("Content-Length", fmt.Sprint(r.size))
	return len(b), nil
}

// Send delays the response to handle Cache-Status
func (r *CustomWriter) Send() (int, error) {
	r.Response.Header.Del("X-Souin-Stored-TTL")
	defer r.Buf.Reset()
	b := esi.Parse(r.Buf.Bytes(), r.Req)
	for h, v := range r.Response.Header {
		if len(v) > 0 {
			r.Rw.Header().Set(h, strings.Join(v, ", "))
		}
	}

	if !r.headersSent {
		r.Rw.Header().Set("Content-Length", fmt.Sprintf("%d", len(b)))
		r.Rw.WriteHeader(r.Response.StatusCode)
	}
	return r.Rw.Write(b)
}

func HasMutation(req *http.Request, rw http.ResponseWriter) bool {
	if req.Context().Value(context.IsMutationRequest).(bool) {
		rfc.MissCache(rw.Header().Set, req, "IS-MUTATION-REQUEST")
		return true
	}
	return false
}

// CanHandle detect if the request can be handled by Souin
func CanHandle(r *http.Request, re types.RetrieverResponsePropertiesInterface) bool {
	co := r.Context().Value(context.RequestCacheControl).(*cacheobject.RequestCacheDirectives)
	return r.Context().Value(context.SupportedMethod).(bool) && co != nil && !co.NoCache && r.Header.Get("Upgrade") != "websocket" && (re.GetExcludeRegexp() == nil || !re.GetExcludeRegexp().MatchString(r.RequestURI))
}

func sendAnyCachedResponse(rh http.Header, response *http.Response, res http.ResponseWriter) {
	response.Header = rh
	for k, v := range response.Header {
		res.Header().Set(k, v[0])
	}
	res.WriteHeader(response.StatusCode)
	b, _ := io.ReadAll(response.Body)
	_, _ = res.Write(b)
	cw, success := res.(souinWriterInterface)
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
	retriever.GetConfiguration().GetLogger().Sugar().Debugf("Incoming request: %+v", req)
	prometheus.Increment(prometheus.RequestCounter)
	start := time.Now()
	cacheCandidate := !strings.Contains(req.Header.Get("Cache-Control"), "no-cache")
	cacheKey := req.Context().Value(context.Key).(string)
	retriever.SetMatchedURLFromRequest(req)
	timeoutCache := req.Context().Value(context.TimeoutCache).(time.Duration)
	cancel := req.Context().Value(context.TimeoutCancel).(ctx.CancelFunc)
	defer cancel()
	foundEntry := make(chan *http.Response)
	errorCacheCh := make(chan error)
	closerCh := make(chan bool)

	go func(isCandidate bool, ret types.RetrieverResponsePropertiesInterface, ckey string) {
		execOnce := make(chan bool, 1)
		execOnce <- true
		for {
			select {
			case <-execOnce:
				if isCandidate {
					r, stale, err := rfc.CachedResponse(
						ret.GetProvider(),
						req,
						ckey,
						ret.GetTransport(),
					)
					if err != nil {
						ret.GetConfiguration().GetLogger().Sugar().Debugf("An error ocurred while retrieving the (stale)? key %s: %v", ckey, err)
						foundEntry <- nil
						errorCacheCh <- err

						return
					}

					if r != nil {
						rh := r.Header
						if stale {
							rfc.HitStaleCache(&rh, ret.GetMatchedURL().TTL.Duration)
							r.Header = rh
						} else {
							prometheus.Increment(prometheus.CachedResponseCounter)
						}
						foundEntry <- r
						errorCacheCh <- nil

						return
					}
				}
				foundEntry <- nil
				errorCacheCh <- nil
			case <-closerCh:
				return
			}
		}
	}(cacheCandidate, retriever, cacheKey)

	defer func() {
		close(errorCacheCh)
		close(foundEntry)
		close(closerCh)
	}()

	select {
	case entry := <-foundEntry:
		if e = <-errorCacheCh; e != nil {
			fmt.Println(e)
			return e
		}
		if entry != nil {
			retriever.GetConfiguration().GetLogger().Sugar().Debugf("Serve response from the cache: %+v", entry)
			sendAnyCachedResponse(entry.Header, entry, res)

			return
		}
	case <-time.After(timeoutCache):
		closerCh <- true
	}

	// coalesceable := make(chan bool)
	errorBackendCh := make(chan error)
	// defer close(coalesceable)
	// go func() {
	// 	defer func() {
	// 		_ = recover()
	// 	}()
	// 	coalesceable <- retriever.GetTransport().GetCoalescingLayerStorage().Exists(cacheKey)
	// }()
	prometheus.Increment(prometheus.NoCachedResponseCounter)

	go func(rs http.ResponseWriter, rq *http.Request) {
		if rc != nil /*&& <-coalesceable*/ {
			rc.Temporize(req, rs, nextMiddleware)
		} else {
			errorBackendCh <- nextMiddleware(rs, rq)
			return
		}
		errorBackendCh <- nil
	}(res, req)

	prometheus.Add(prometheus.AvgResponseTime, float64(time.Since(start).Milliseconds()))

	select {
	case <-req.Context().Done():
		switch req.Context().Err() {
		case ctx.DeadlineExceeded:
			cw := res.(*CustomWriter)
			rfc.MissCache(cw.Header().Set, req, "DEADLINE-EXCEEDED")
			cw.WriteHeader(http.StatusGatewayTimeout)
			_, _ = cw.Rw.Write(serverTimeoutMessage)
			return ctx.DeadlineExceeded
		case ctx.Canceled:
			return nil
		default:
			return nil
		}
	case v := <-errorBackendCh:
		if v == nil {
			_, _ = res.(souinWriterInterface).Send()
		}
		return v
	}
}

// DefaultSouinPluginInitializerFromConfiguration is the default initialization for plugins
func DefaultSouinPluginInitializerFromConfiguration(c configurationtypes.AbstractConfigurationInterface) *types.RetrieverResponseProperties {
	if c.GetLogger() == nil {
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
	}

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
