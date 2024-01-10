package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/pkg/api"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/surrogate"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/pquerna/cachecontrol/cacheobject"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewHTTPCacheHandler(c configurationtypes.AbstractConfigurationInterface) *SouinBaseHandler {
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

	storers, err := storage.NewStorages(c)
	if err != nil {
		panic(err)
	}
	c.GetLogger().Debug("Storer initialized.")
	regexpUrls := helpers.InitializeRegexp(c)
	surrogateStorage := surrogate.InitializeSurrogate(c)
	c.GetLogger().Debug("Surrogate storage initialized.")
	var excludedRegexp *regexp.Regexp = nil
	if c.GetDefaultCache().GetRegex().Exclude != "" {
		excludedRegexp = regexp.MustCompile(c.GetDefaultCache().GetRegex().Exclude)
	}

	ctx := context.GetContext()
	ctx.Init(c)

	bufPool := &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	defaultMatchedUrl := configurationtypes.URL{
		TTL:                 configurationtypes.Duration{Duration: c.GetDefaultCache().GetTTL()},
		Headers:             c.GetDefaultCache().GetHeaders(),
		DefaultCacheControl: c.GetDefaultCache().GetDefaultCacheControl(),
	}
	c.GetLogger().Info("Souin configuration is now loaded.")

	return &SouinBaseHandler{
		Configuration:            c,
		Storers:                  storers,
		InternalEndpointHandlers: api.GenerateHandlerMap(c, storers, surrogateStorage),
		ExcludeRegex:             excludedRegexp,
		RegexpUrls:               regexpUrls,
		DefaultMatchedUrl:        defaultMatchedUrl,
		SurrogateKeyStorer:       surrogateStorage,
		context:                  ctx,
		bufPool:                  bufPool,
		storersLen:               len(storers),
	}
}

type SouinBaseHandler struct {
	Configuration            configurationtypes.AbstractConfigurationInterface
	Storers                  []storage.Storer
	InternalEndpointHandlers *api.MapHandler
	ExcludeRegex             *regexp.Regexp
	RegexpUrls               regexp.Regexp
	SurrogateKeys            configurationtypes.SurrogateKeys
	SurrogateKeyStorer       providers.SurrogateInterface
	DefaultMatchedUrl        configurationtypes.URL
	context                  *context.Context
	bufPool                  *sync.Pool
	storersLen               int
}

type upsreamError struct{}

func (upsreamError) Error() string {
	return "Upstream error"
}

func (s *SouinBaseHandler) Upstream(
	customWriter *CustomWriter,
	rq *http.Request,
	next handlerFunc,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
) error {
	now := time.Now().UTC()
	rq.Header.Set("Date", now.Format(time.RFC1123))
	if err := next(customWriter, rq); err != nil {
		customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=SERVE-HTTP-ERROR", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
		return err
	}

	switch customWriter.statusCode {
	case 500, 502, 503, 504:
		return new(upsreamError)
	}

	responseCc, _ := cacheobject.ParseResponseCacheControl(customWriter.Header().Get("Cache-Control"))

	currentMatchedURL := s.DefaultMatchedUrl
	if regexpURL := s.RegexpUrls.FindString(rq.Host + rq.URL.Path); regexpURL != "" {
		u := s.Configuration.GetUrls()[regexpURL]
		if u.TTL.Duration != 0 {
			currentMatchedURL.TTL = u.TTL
		}
		if len(u.Headers) != 0 {
			currentMatchedURL.Headers = u.Headers
		}
	}

	ma := currentMatchedURL.TTL.Duration
	if responseCc.MaxAge > 0 {
		ma = time.Duration(responseCc.MaxAge) * time.Second
	} else if responseCc.SMaxAge > 0 {
		ma = time.Duration(responseCc.SMaxAge) * time.Second
	}
	if ma > currentMatchedURL.TTL.Duration {
		ma = currentMatchedURL.TTL.Duration
	}
	date, _ := http.ParseTime(now.Format(http.TimeFormat))
	customWriter.Headers.Set(rfc.StoredTTLHeader, ma.String())
	ma = ma - time.Since(date)

	status := fmt.Sprintf("%s; fwd=uri-miss", rq.Context().Value(context.CacheName))
	if !requestCc.NoStore && !responseCc.NoStore {
		res := http.Response{
			StatusCode: customWriter.statusCode,
			Body:       io.NopCloser(bytes.NewBuffer(customWriter.Buf.Bytes())),
			Header:     customWriter.Headers,
		}

		res.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		res.Request = rq
		response, err := httputil.DumpResponse(&res, true)
		if err == nil {
			variedHeaders, isVaryStar := rfc.VariedHeaderAllCommaSepValues(res.Header)
			if isVaryStar {
				// "Implies that the response is uncacheable"
				status += "; detail=UPSTREAM-VARY-STAR"
			} else {
				cachedKey += rfc.GetVariedCacheKey(rq, variedHeaders)
				var wg sync.WaitGroup
				mu := sync.Mutex{}
				fails := []string{}
				select {
				case <-rq.Context().Done():
					status += "; detail=REQUEST-CANCELED-OR-UPSTREAM-BROKEN-PIPE"
				default:
					for _, storer := range s.Storers {
						wg.Add(1)
						go func(currentStorer storage.Storer) {
							defer wg.Done()
							if currentStorer.Set(cachedKey, response, currentMatchedURL, ma) != nil {
								mu.Lock()
								fails = append(fails, fmt.Sprintf("; detail=%s-INSERTION-ERROR", currentStorer.Name()))
								mu.Unlock()
							}
						}(storer)
					}

					wg.Wait()
					if len(fails) < len(s.Storers) {
						go func(rs http.Response, key string) {
							_ = s.SurrogateKeyStorer.Store(&rs, key)
						}(res, cachedKey)
						status += "; stored"
					}

					if len(fails) > 0 {
						status += strings.Join(fails, "")
					}
				}
			}
		}
	} else {
		status += "; detail=NO-STORE-DIRECTIVE"
	}
	customWriter.Headers.Set("Cache-Status", status+"; key="+rfc.GetCacheKeyFromCtx(rq.Context()))

	return nil
}

func (s *SouinBaseHandler) HandleInternally(r *http.Request) (bool, http.HandlerFunc) {
	if s.InternalEndpointHandlers != nil {
		for k, handler := range *s.InternalEndpointHandlers.Handlers {
			if strings.Contains(r.RequestURI, k) {
				return true, handler
			}
		}
	}

	return false, nil
}

type handlerFunc = func(http.ResponseWriter, *http.Request) error

func (s *SouinBaseHandler) ServeHTTP(rw http.ResponseWriter, baseRq *http.Request, next handlerFunc) error {
	if b, handler := s.HandleInternally(baseRq); b {
		handler(rw, baseRq)
		return nil
	}

	rq := s.context.SetBaseContext(baseRq)
	cacheName := rq.Context().Value(context.CacheName).(string)
	if rq.Header.Get("Upgrade") == "websocket" || (s.ExcludeRegex != nil && s.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=EXCLUDED-REQUEST-URI")
		return next(rw, rq)
	}

	if !rq.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=UNSUPPORTED-METHOD")

		return next(rw, rq)
	}

	requestCc, coErr := cacheobject.ParseRequestCacheControl(rq.Header.Get("Cache-Control"))

	if coErr != nil || requestCc == nil {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		return next(rw, rq)
	}

	rq = s.context.SetContext(rq, baseRq)
	cachedKey := rq.Context().Value(context.Key).(string)

	bufPool := s.bufPool.Get().(*bytes.Buffer)
	bufPool.Reset()
	defer s.bufPool.Put(bufPool)
	customWriter := NewCustomWriter(rq, rw, bufPool)
	if !requestCc.NoCache {
		validator := rfc.ParseRequest(rq)
		var response *http.Response
		for _, currentStorer := range s.Storers {
			response = currentStorer.Prefix(cachedKey, rq, validator)
			if response != nil {
				s.Configuration.GetLogger().Sugar().Debugf("Found response in the %s storage", currentStorer.Name())
				break
			}
		}

		if response != nil && rfc.ValidateCacheControl(response, requestCc) {
			rfc.SetCacheStatusHeader(response)
			if rfc.ValidateMaxAgeCachedResponse(requestCc, response) != nil {
				customWriter.Headers = response.Header
				customWriter.statusCode = response.StatusCode
				io.Copy(customWriter.Buf, response.Body)
				customWriter.Send()

				return nil
			}
		} else if response == nil && (requestCc.MaxStaleSet || requestCc.MaxStale > -1) {
			for _, currentStorer := range s.Storers {
				response = currentStorer.Prefix(storage.StalePrefix+cachedKey, rq, validator)
				if response != nil {
					break
				}
			}
			if nil != response && rfc.ValidateCacheControl(response, requestCc) {
				addTime, _ := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
				rfc.SetCacheStatusHeader(response)

				responseCc, _ := cacheobject.ParseResponseCacheControl(response.Header.Get("Cache-Control"))
				if responseCc.StaleWhileRevalidate > 0 {
					customWriter.Headers = response.Header
					customWriter.statusCode = response.StatusCode
					rfc.HitStaleCache(&response.Header)
					io.Copy(customWriter.Buf, response.Body)
					customWriter.Send()
					customWriter = NewCustomWriter(rq, rw, bufPool)
					go s.Upstream(customWriter, rq, next, requestCc, cachedKey)
					buf := s.bufPool.Get().(*bytes.Buffer)
					buf.Reset()
					defer s.bufPool.Put(buf)

					return nil
				}

				if responseCc.StaleIfError > 0 && s.Upstream(customWriter, rq, next, requestCc, cachedKey) != nil {
					customWriter.Headers = response.Header
					customWriter.statusCode = response.StatusCode
					rfc.HitStaleCache(&response.Header)
					io.Copy(customWriter.Buf, response.Body)
					customWriter.Send()

					return nil
				}

				if rfc.ValidateMaxAgeCachedStaleResponse(requestCc, response, int(addTime.Seconds())) != nil {
					customWriter.Headers = response.Header
					customWriter.statusCode = response.StatusCode
					rfc.HitStaleCache(&response.Header)
					io.Copy(customWriter.Buf, response.Body)
					customWriter.Send()

					return nil
				}
			}
		}
	}

	if err := s.Upstream(customWriter, rq, next, requestCc, cachedKey); err != nil {
		return err
	}

	_, _ = customWriter.Send()
	return nil
}
