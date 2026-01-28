package middleware

import (
	"bytes"
	baseCtx "context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/helpers"
	"github.com/darkweak/souin/pkg/api"
	"github.com/darkweak/souin/pkg/api/prometheus"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/darkweak/storages/core"
	"github.com/google/uuid"
	"github.com/pquerna/cachecontrol/cacheobject"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/singleflight"
)

func reorderStorers(storers []types.Storer, expectedStorers []string) []types.Storer {
	if len(expectedStorers) == 0 {
		return storers
	}

	newStorers := make([]types.Storer, 0)
	for _, expectedStorer := range expectedStorers {
		for _, storer := range storers {
			if storer.Name() == strings.ToUpper(expectedStorer) {
				newStorers = append(newStorers, storer)
			}
		}
	}

	return newStorers
}

const (
	evictionLockKey = "eviction-lock"
	evictionLockTTL = 2 * time.Minute
)

// evictionLockHolder is a unique identifier for this instance, used for distributed lock ownership.
var evictionLockHolder = uuid.NewString()

func tryAcquireEvictionLock(storer types.Storer) bool {
	now := time.Now()
	existing := storer.Get(evictionLockKey)

	if len(existing) > 0 {
		// Lock value format: "holder_id|expiry_timestamp"
		parts := strings.SplitN(string(existing), "|", 2)
		if len(parts) == 2 {
			holderID := parts[0]
			lockedUntil, err := time.Parse(time.RFC3339, parts[1])
			if err == nil && now.Before(lockedUntil) {
				// Lock is still valid - check if we own it
				if holderID == evictionLockHolder {
					return true
				}
				return false
			}
		}
	}

	// Lock expired or doesn't exist - attempt to claim it using optimistic locking
	newLockExpiry := now.Add(evictionLockTTL)
	lockValue := evictionLockHolder + "|" + newLockExpiry.Format(time.RFC3339)
	if err := storer.Set(evictionLockKey, []byte(lockValue), evictionLockTTL); err != nil {
		return false
	}

	// Verify we actually got the lock (optimistic locking)
	// Another instance might have written between our check and set
	time.Sleep(10 * time.Millisecond)
	verifyValue := storer.Get(evictionLockKey)
	return string(verifyValue) == lockValue
}

func registerMappingKeysEviction(logger core.Logger, storers []types.Storer, interval time.Duration) {
	for _, storer := range storers {
		logger.Debugf("registering mapping eviction for storer %s (interval: %s)", storer.Name(), interval)
		go func(current types.Storer) {
			for {
				if !tryAcquireEvictionLock(current) {
					logger.Debugf("skipping mapping eviction for storer %s, another instance holds the lock", current.Name())
					time.Sleep(time.Minute)
					continue
				}

				logger.Debugf("run mapping eviction for storer %s", current.Name())
				api.EvictMapping(current)
				<-time.After(interval)
			}
		}(storer)
	}
}

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
		c.SetLogger(logger.Sugar())
	}

	storedStorers := core.GetRegisteredStorers()
	storers := []types.Storer{}
	if len(storedStorers) != 0 {
		dc := c.GetDefaultCache()
		for _, s := range []string{dc.GetBadger().Uuid, dc.GetEtcd().Uuid, dc.GetNats().Uuid, dc.GetNuts().Uuid, dc.GetOlric().Uuid, dc.GetOtter().Uuid, dc.GetRedis().Uuid, dc.GetSimpleFS().Uuid} {
			if s != "" {
				if st := core.GetRegisteredStorer(s); st != nil {
					storers = append(storers, st.(types.Storer))
				}
			}
		}

		storers = reorderStorers(storers, c.GetDefaultCache().GetStorers())

		if len(storers) > 0 {
			names := []string{}
			for _, storer := range storers {
				names = append(names, storer.Name())
			}
			c.GetLogger().Debugf("You're running Souin with the following storages in this order %s", strings.Join(names, ", "))
		}
	}
	if len(storers) == 0 {
		c.GetLogger().Warn("You're running Souin with the default storage that is not optimized and for development purpose. We recommend to use at least one of the storages from https://github.com/darkweak/storages")

		memoryStorer, _ := storage.Factory(c)
		if st := core.GetRegisteredStorer(types.DefaultStorageName + "-"); st != nil {
			memoryStorer = st.(types.Storer)
		} else {
			core.RegisterStorage(memoryStorer)
		}
		storers = append(storers, memoryStorer)
	}

	c.GetLogger().Debugf("Storer initialized: %#v.", storers)
	regexpUrls := helpers.InitializeRegexp(c)
	surrogateStorage := surrogate.InitializeSurrogate(c, fmt.Sprintf("%s-%s", storers[0].Name(), storers[0].Uuid()))
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

	registerMappingKeysEviction(c.GetLogger(), storers, c.GetDefaultCache().GetMappingEvictionInterval())

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
		singleflightPool:         singleflight.Group{},
	}
}

type SouinBaseHandler struct {
	Configuration            configurationtypes.AbstractConfigurationInterface
	Storers                  []types.Storer
	InternalEndpointHandlers *api.MapHandler
	ExcludeRegex             *regexp.Regexp
	RegexpUrls               regexp.Regexp
	SurrogateKeys            configurationtypes.SurrogateKeys
	SurrogateKeyStorer       providers.SurrogateInterface
	DefaultMatchedUrl        configurationtypes.URL
	context                  *context.Context
	singleflightPool         singleflight.Group
	bufPool                  *sync.Pool
	storersLen               int
}

var Upstream50xError = upstream50xError{}

type upstream50xError struct{}

func (upstream50xError) Error() string {
	return "Upstream 50x error"
}

func isCacheableCode(code int) bool {
	switch code {
	case 200, 203, 204, 206, 300, 301, 404, 405, 410, 414, 501:
		return true
	}

	return false
}

func canStatusCodeEmptyContent(code int) bool {
	switch code {
	case 204, 301, 405:
		return true
	}

	return false
}

func canBypassAuthorizationRestriction(headers http.Header, bypassed []string) bool {
	for _, header := range bypassed {
		if strings.ToLower(header) == "authorization" {
			return true
		}
	}

	return strings.Contains(strings.ToLower(headers.Get("Vary")), "authorization")
}

func (s *SouinBaseHandler) hasAllowedAdditionalStatusCodesToCache(code int) bool {
	for _, sc := range s.Configuration.GetDefaultCache().GetAllowedAdditionalStatusCodes() {
		if sc == code {
			return true
		}
	}

	return false
}

func (s *SouinBaseHandler) Store(
	customWriter *CustomWriter,
	rq *http.Request,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
) error {
	statusCode := customWriter.GetStatusCode()
	if !isCacheableCode(statusCode) && !s.hasAllowedAdditionalStatusCodesToCache(statusCode) {
		customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=UNCACHEABLE-STATUS-CODE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

		switch statusCode {
		case 500, 502, 503, 504:
			return Upstream50xError
		}

		return nil
	}

	headerName, cacheControl := s.SurrogateKeyStorer.GetSurrogateControl(customWriter.Header())
	if cacheControl == "" {
		// TODO see with @mnot if mandatory to not store the response when no Cache-Control given.
		// if s.DefaultMatchedUrl.DefaultCacheControl == "" {
		// 	customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=EMPTY-RESPONSE-CACHE-CONTROL", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
		// 	return nil
		// }
		customWriter.Header().Set(headerName, s.DefaultMatchedUrl.DefaultCacheControl)
	}

	responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(customWriter.Header(), headerName))
	s.Configuration.GetLogger().Debugf("Response cache-control %+v", responseCc)
	if responseCc == nil {
		customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=INVALID-RESPONSE-CACHE-CONTROL", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
		return nil
	}

	modeContext := rq.Context().Value(context.Mode).(*context.ModeContext)
	if !modeContext.Bypass_request && (responseCc.PrivatePresent || rq.Header.Get("Authorization") != "") && !canBypassAuthorizationRestriction(customWriter.Header(), rq.Context().Value(context.IgnoredHeaders).([]string)) {
		customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=PRIVATE-OR-AUTHENTICATED-RESPONSE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
		return nil
	}

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

	hasFreshness := false
	ma := currentMatchedURL.TTL.Duration
	if !modeContext.Bypass_response {
		if responseCc.SMaxAge >= 0 {
			ma = time.Duration(responseCc.SMaxAge) * time.Second
		} else if responseCc.MaxAge >= 0 {
			ma = time.Duration(responseCc.MaxAge) * time.Second
		} else if customWriter.Header().Get("Expires") != "" {
			exp, err := time.Parse(time.RFC1123, customWriter.Header().Get("Expires"))
			if err != nil {
				return nil
			}

			duration := time.Until(exp)
			if duration <= 0 || duration > 10*types.OneYearDuration {
				return nil
			}

			date, _ := time.Parse(time.RFC1123, customWriter.Header().Get("Date"))
			if date.Sub(exp) > 0 {
				return nil
			}

			ma = duration
			hasFreshness = true
		}
	}

	now := rq.Context().Value(context.Now).(time.Time)
	date, _ := http.ParseTime(now.Format(http.TimeFormat))
	customWriter.Header().Set(rfc.StoredTTLHeader, ma.String())
	ma = ma - time.Since(date)

	status := fmt.Sprintf("%s; fwd=uri-miss", rq.Context().Value(context.CacheName))
	if (modeContext.Bypass_request || !requestCc.NoStore) &&
		(modeContext.Bypass_response || !responseCc.NoStore || hasFreshness) {
		headers := customWriter.Header().Clone()
		for hname, shouldDelete := range responseCc.NoCache {
			if shouldDelete {
				headers.Del(hname)
			}
		}

		customWriter.mutex.Lock()
		b := customWriter.Buf.Bytes()
		bLen := customWriter.Buf.Len()
		customWriter.mutex.Unlock()

		res := http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewBuffer(b)),
			Header:     headers,
		}

		if res.Header.Get("Date") == "" {
			res.Header.Set("Date", now.Format(http.TimeFormat))
		}
		if res.Header.Get("Content-Length") == "" {
			res.Header.Set("Content-Length", fmt.Sprint(bLen))
		}
		respBodyMaxSize := int(s.Configuration.GetDefaultCache().GetMaxBodyBytes())
		if respBodyMaxSize > 0 && bLen > respBodyMaxSize {
			customWriter.Header().Set("Cache-Status", status+"; detail=UPSTREAM-RESPONSE-TOO-LARGE; key="+rfc.GetCacheKeyFromCtx(rq.Context()))

			return nil
		}
		res.Header.Set(rfc.StoredLengthHeader, res.Header.Get("Content-Length"))
		response, err := httputil.DumpResponse(&res, true)
		if err == nil && (bLen > 0 || rq.Method == http.MethodHead || canStatusCodeEmptyContent(statusCode) || s.hasAllowedAdditionalStatusCodesToCache(statusCode)) {
			variedHeaders, isVaryStar := rfc.VariedHeaderAllCommaSepValues(res.Header)
			if isVaryStar {
				// "Implies that the response is uncacheable"
				status += "; detail=UPSTREAM-VARY-STAR"
			} else {
				variedKey := cachedKey + rfc.GetVariedCacheKey(rq, variedHeaders)
				if rq.Context().Value(context.Hashed).(bool) {
					cachedKey = fmt.Sprint(xxhash.Sum64String(cachedKey))
					variedKey = fmt.Sprint(xxhash.Sum64String(variedKey))
				}
				s.Configuration.GetLogger().Debugf("Store the response for %s with duration %v", variedKey, ma)

				var wg sync.WaitGroup
				mu := sync.Mutex{}
				fails := []string{}
				select {
				case <-rq.Context().Done():
					status += "; detail=REQUEST-CANCELED-OR-UPSTREAM-BROKEN-PIPE"
				default:
					vhs := http.Header{}
					for _, hname := range variedHeaders {
						hn := strings.Split(hname, ":")
						vhs.Set(hn[0], rq.Header.Get(hn[0]))
					}
					if upstreamStorerTarget := res.Header.Get("X-Souin-Storer"); upstreamStorerTarget != "" {
						res.Header.Del("X-Souin-Storer")

						var overridedStorer types.Storer
						for _, storer := range s.Storers {
							if strings.Contains(strings.ToLower(storer.Name()), strings.ToLower(upstreamStorerTarget)) {
								overridedStorer = storer
							}
						}

						if overridedStorer.SetMultiLevel(
							cachedKey,
							variedKey,
							response,
							vhs,
							res.Header.Get("Etag"), ma,
							variedKey,
						) == nil {
							s.Configuration.GetLogger().Debugf("Stored the key %s in the %s provider", variedKey, overridedStorer.Name())
							res.Request = rq
						} else {
							fails = append(fails, fmt.Sprintf("; detail=%s-INSERTION-ERROR", overridedStorer.Name()))
						}
					} else {
						for _, storer := range s.Storers {
							wg.Add(1)
							go func(currentStorer types.Storer, currentRes http.Response) {
								defer wg.Done()
								if currentStorer.SetMultiLevel(
									cachedKey,
									variedKey,
									response,
									vhs,
									currentRes.Header.Get("Etag"), ma,
									variedKey,
								) == nil {
									s.Configuration.GetLogger().Debugf("Stored the key %s in the %s provider", variedKey, currentStorer.Name())
									currentRes.Request = rq
								} else {
									mu.Lock()
									fails = append(fails, fmt.Sprintf("; detail=%s-INSERTION-ERROR", currentStorer.Name()))
									mu.Unlock()
								}
							}(storer, res)
						}
					}

					wg.Wait()
					if len(fails) < s.storersLen {
						if !s.Configuration.IsSurrogateDisabled() {
							go func(rs http.Response, key string) {
								_ = s.SurrogateKeyStorer.Store(&rs, key, uri)
							}(res, variedKey)
						}

						status += "; stored"
					}

					if len(fails) > 0 {
						status += strings.Join(fails, "")
					}
				}
			}

		} else {
			status += "; detail=UPSTREAM-ERROR-OR-EMPTY-RESPONSE"
		}
	} else {
		status += "; detail=NO-STORE-DIRECTIVE"
	}
	customWriter.Header().Set("Cache-Status", status+"; key="+rfc.GetCacheKeyFromCtx(rq.Context()))

	return nil
}

type singleflightValue struct {
	body              []byte
	headers           http.Header
	requestHeaders    http.Header
	code              int
	disableCoalescing bool
}

func (s *SouinBaseHandler) Upstream(
	customWriter *CustomWriter,
	rq *http.Request,
	next handlerFunc,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
	disableCoalescing bool,
) error {
	s.Configuration.GetLogger().Debug("Request the upstream server")
	prometheus.Increment(prometheus.RequestCounter)

	var recoveredFromErr error = nil
	defer func() {
		// In case of "http.ErrAbortHandler" panic,
		// prevent singleflight from wrapping it into "singleflight.panicError".
		if r := recover(); r != nil {
			err, ok := r.(error)
			// Sometimes, the error is a string.
			if !ok || errors.Is(err, http.ErrAbortHandler) {
				recoveredFromErr = http.ErrAbortHandler
			} else {
				panic(err)
			}
		}
	}()

	singleflightCacheKey := cachedKey
	if s.Configuration.GetDefaultCache().IsCoalescingDisable() || disableCoalescing {
		singleflightCacheKey += uuid.NewString()
	}
	sfValue, err, shared := s.singleflightPool.Do(singleflightCacheKey, func() (interface{}, error) {
		if e := next(customWriter, rq); e != nil {
			s.Configuration.GetLogger().Warnf("%#v", e)
			customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=SERVE-HTTP-ERROR", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
			return nil, e
		}

		if !s.Configuration.IsSurrogateDisabled() {
			s.SurrogateKeyStorer.Invalidate(rq.Method, customWriter.Header())
		}

		statusCode := customWriter.GetStatusCode()
		if !isCacheableCode(statusCode) && !s.hasAllowedAdditionalStatusCodesToCache(statusCode) {
			customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=UNCACHEABLE-STATUS-CODE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

			switch statusCode {
			case 500, 502, 503, 504:
				return nil, Upstream50xError
			}
		}

		headerName, cacheControl := s.SurrogateKeyStorer.GetSurrogateControl(customWriter.Header())
		if cacheControl == "" {
			customWriter.Header().Set(headerName, s.DefaultMatchedUrl.DefaultCacheControl)
		}

		err := s.Store(customWriter, rq, requestCc, cachedKey, uri)
		defer customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})

		return singleflightValue{
			body:              customWriter.Buf.Bytes(),
			headers:           customWriter.Header().Clone(),
			requestHeaders:    rq.Header,
			code:              statusCode,
			disableCoalescing: strings.Contains(cacheControl, "private") || customWriter.Header().Get("Set-Cookie") != "",
		}, err
	})
	if recoveredFromErr != nil {
		panic(recoveredFromErr)
	}
	if err != nil {
		return err
	}

	if sfWriter, ok := sfValue.(singleflightValue); ok {
		if shared && sfWriter.disableCoalescing {
			return s.Upstream(customWriter, rq, next, requestCc, cachedKey, uri, true)
		}

		if vary := sfWriter.headers.Get("Vary"); vary != "" {
			variedHeaders, isVaryStar := rfc.VariedHeaderAllCommaSepValues(sfWriter.headers)
			if !isVaryStar {
				for _, vh := range variedHeaders {
					if rq.Header.Get(vh) != sfWriter.requestHeaders.Get(vh) {
						// cachedKey += rfc.GetVariedCacheKey(rq, variedHeaders)
						return s.Upstream(customWriter, rq, next, requestCc, cachedKey, uri, false)
					}
				}
			}
		}

		if shared {
			s.Configuration.GetLogger().Infof("Reused response from concurrent request with the key %s", cachedKey)
		}
		_, _ = customWriter.Write(sfWriter.body)
		maps.Copy(customWriter.Header(), sfWriter.headers)
		customWriter.WriteHeader(sfWriter.code)
	}

	return nil
}

func (s *SouinBaseHandler) Revalidate(validator *core.Revalidator, next handlerFunc, customWriter *CustomWriter, rq *http.Request, requestCc *cacheobject.RequestCacheDirectives, cachedKey string, uri string) error {
	s.Configuration.GetLogger().Debug("Revalidate the request with the upstream server")
	prometheus.Increment(prometheus.RequestRevalidationCounter)

	singleflightCacheKey := cachedKey
	if s.Configuration.GetDefaultCache().IsCoalescingDisable() {
		singleflightCacheKey += uuid.NewString()
	}
	sfValue, err, shared := s.singleflightPool.Do(singleflightCacheKey, func() (interface{}, error) {
		err := next(customWriter, rq)

		if !s.Configuration.IsSurrogateDisabled() {
			s.SurrogateKeyStorer.Invalidate(rq.Method, customWriter.Header())
		}

		statusCode := customWriter.GetStatusCode()
		if err == nil {
			if validator.IfUnmodifiedSincePresent && statusCode != http.StatusNotModified {
				customWriter.handleBuffer(func(b *bytes.Buffer) {
					b.Reset()
				})
				customWriter.Rw.WriteHeader(http.StatusPreconditionFailed)

				return nil, errors.New("")
			}

			if validator.IfModifiedSincePresent {
				if lastModified, err := time.Parse(time.RFC1123, customWriter.Header().Get("Last-Modified")); err == nil && validator.IfModifiedSince.Sub(lastModified) > 0 {
					customWriter.handleBuffer(func(b *bytes.Buffer) {
						b.Reset()
					})
					customWriter.Rw.WriteHeader(http.StatusNotModified)

					return nil, errors.New("")
				}
			}

			if statusCode != http.StatusNotModified {
				err = s.Store(customWriter, rq, requestCc, cachedKey, uri)
			}
		}

		customWriter.Header().Set(
			"Cache-Status",
			fmt.Sprintf(
				"%s; fwd=request; fwd-status=%d; key=%s; detail=REQUEST-REVALIDATION",
				rq.Context().Value(context.CacheName),
				statusCode,
				rfc.GetCacheKeyFromCtx(rq.Context()),
			),
		)

		defer customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})
		return singleflightValue{
			body:    customWriter.Buf.Bytes(),
			headers: customWriter.Header().Clone(),
			code:    statusCode,
		}, err
	})

	if sfWriter, ok := sfValue.(singleflightValue); ok {
		if shared {
			s.Configuration.GetLogger().Infof("Reused response from concurrent request with the key %s", cachedKey)
		}
		_, _ = customWriter.Write(sfWriter.body)
		maps.Copy(customWriter.Header(), sfWriter.headers)
		customWriter.WriteHeader(sfWriter.code)
	}

	return err
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
type statusCodeLogger struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusCodeLogger) WriteHeader(code int) {
	s.statusCode = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *SouinBaseHandler) backfillStorers(idx int, cachedKey string, rq *http.Request, response *http.Response) {
	if idx == 0 {
		return
	}

	storedDuration, err := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
	if err != nil {
		return
	}

	dateHeader, err := http.ParseTime(response.Header.Get("Date"))
	if err != nil {
		return
	}

	now := time.Now()

	ma := storedDuration - now.Sub(dateHeader)

	variedHeaders, _ := rfc.VariedHeaderAllCommaSepValues(response.Header)
	variedKey := cachedKey + rfc.GetVariedCacheKey(rq, variedHeaders)

	if rq.Context().Value(context.Hashed).(bool) {
		cachedKey = fmt.Sprint(xxhash.Sum64String(cachedKey))
		variedKey = fmt.Sprint(xxhash.Sum64String(variedKey))
	}

	vhs := http.Header{}
	for _, hname := range variedHeaders {
		hn := strings.Split(hname, ":")
		vhs.Set(hn[0], rq.Header.Get(hn[0]))
	}

	res, _ := httputil.DumpResponse(response, true)

	for _, currentStorer := range s.Storers[:idx] {
		err = currentStorer.SetMultiLevel(
			cachedKey,
			variedKey,
			res,
			vhs,
			response.Header.Get("Etag"), ma,
			variedKey,
		)
		if err != nil {
			s.Configuration.GetLogger().Errorf("Error while backfilling the storer %s: %v", currentStorer.Name(), err)
		}
	}
}

func (s *SouinBaseHandler) ServeHTTP(rw http.ResponseWriter, rq *http.Request, next handlerFunc) error {
	start := time.Now()
	defer func(s time.Time) {
		prometheus.Add(prometheus.AvgResponseTime, float64(time.Since(s).Milliseconds()))
	}(start)
	s.Configuration.GetLogger().Debugf("Incomming request %+v", rq)
	if b, handler := s.HandleInternally(rq); b {
		handler(rw, rq)
		return nil
	}

	req := s.context.SetBaseContext(rq)
	cacheName := req.Context().Value(context.CacheName).(string)

	if rq.Header.Get("Upgrade") == "websocket" || rq.Header.Get("Accept") == "text/event-stream" || (s.ExcludeRegex != nil && s.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=EXCLUDED-REQUEST-URI")
		return next(rw, req)
	}

	if !req.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=UNSUPPORTED-METHOD")
		nrw := &statusCodeLogger{
			ResponseWriter: rw,
			statusCode:     0,
		}

		err := next(nrw, req)

		if !s.Configuration.IsSurrogateDisabled() {
			s.SurrogateKeyStorer.Invalidate(req.Method, rw.Header())
		}

		if err == nil && req.Method != http.MethodGet && nrw.statusCode < http.StatusBadRequest {
			// Invalidate related GET keys when the method is not allowed and the response is valid
			req.Method = http.MethodGet
			keyname := s.context.SetContext(req, rq).Context().Value(context.Key).(string)
			for _, storer := range s.Storers {
				storer.Delete(core.MappingKeyPrefix + keyname)
			}
		}

		return err
	}

	requestCc, coErr := cacheobject.ParseRequestCacheControl(rfc.HeaderAllCommaSepValuesString(req.Header, "Cache-Control"))

	modeContext := req.Context().Value(context.Mode).(*context.ModeContext)
	if !modeContext.Bypass_request && (coErr != nil || requestCc == nil) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		err := next(rw, req)

		if !s.Configuration.IsSurrogateDisabled() {
			s.SurrogateKeyStorer.Invalidate(req.Method, rw.Header())
		}

		return err
	}

	req = s.context.SetContext(req, rq)
	if req.Context().Value(context.IsMutationRequest).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=IS-MUTATION-REQUEST")

		err := next(rw, req)

		if !s.Configuration.IsSurrogateDisabled() {
			s.SurrogateKeyStorer.Invalidate(req.Method, rw.Header())
		}

		return err
	}
	cachedKey := req.Context().Value(context.Key).(string)

	// Need to copy URL path before calling next because it can alter the URI
	uri := req.URL.Path
	bufPool := s.bufPool.Get().(*bytes.Buffer)
	bufPool.Reset()
	defer s.bufPool.Put(bufPool)

	customWriter := NewCustomWriter(req, rw, bufPool)
	customWriter.Headers.Add("Range", req.Header.Get("Range"))
	req.Header.Del("Range")

	go func(req *http.Request, crw *CustomWriter) {
		<-req.Context().Done()
		crw.mutex.Lock()
		crw.headersSent = true
		crw.mutex.Unlock()
	}(req, customWriter)

	backfillIds := 0

	s.Configuration.GetLogger().Debugf("Request cache-control %+v", requestCc)
	if modeContext.Bypass_request || !requestCc.NoCache {
		validator := rfc.ParseRequest(req)
		var fresh, stale *http.Response
		var storerName string
		finalKey := cachedKey
		if req.Context().Value(context.Hashed).(bool) {
			finalKey = fmt.Sprint(xxhash.Sum64String(finalKey))
		}
		for _, currentStorer := range s.Storers {
			fresh, stale = currentStorer.GetMultiLevel(finalKey, req, validator)

			if fresh != nil || stale != nil {
				storerName = currentStorer.Name()
				s.Configuration.GetLogger().Debugf("Found at least one valid response in the %s storage", storerName)
				break
			}

			backfillIds++
		}

		headerName, _ := s.SurrogateKeyStorer.GetSurrogateControl(customWriter.Header())
		if fresh != nil && (!modeContext.Strict || rfc.ValidateCacheControl(fresh, requestCc)) {
			go func() {
				s.backfillStorers(backfillIds, cachedKey, req, fresh)
			}()

			response := fresh

			if validator.ResponseETag != "" && validator.Matched {
				rfc.SetCacheStatusHeader(response, storerName)
				for h, v := range response.Header {
					customWriter.Header()[h] = v
				}
				if validator.NotModified {
					customWriter.WriteHeader(http.StatusNotModified)
					customWriter.handleBuffer(func(b *bytes.Buffer) {
						b.Reset()
					})
					_, _ = customWriter.Send()

					return nil
				}

				customWriter.WriteHeader(response.StatusCode)
				customWriter.handleBuffer(func(b *bytes.Buffer) {
					_, _ = io.Copy(b, response.Body)
				})
				_, _ = customWriter.Send()

				return nil
			}

			if !modeContext.Bypass_request && validator.NeedRevalidation {
				err := s.Revalidate(validator, next, customWriter, req, requestCc, cachedKey, uri)
				_, _ = customWriter.Send()

				return err
			}
			if resCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(response.Header, headerName)); !modeContext.Bypass_response && resCc.NoCachePresent {
				prometheus.Increment(prometheus.NoCachedResponseCounter)
				err := s.Revalidate(validator, next, customWriter, req, requestCc, cachedKey, uri)
				_, _ = customWriter.Send()

				return err
			}
			rfc.SetCacheStatusHeader(response, storerName)
			if !modeContext.Strict || rfc.ValidateMaxAgeCachedResponse(requestCc, response) != nil {
				for h, v := range response.Header {
					customWriter.Header()[h] = v
				}
				customWriter.WriteHeader(response.StatusCode)
				s.Configuration.GetLogger().Debugf("Serve from cache %+v", req)
				customWriter.handleBuffer(func(b *bytes.Buffer) {
					_, _ = io.Copy(b, response.Body)
				})
				_, err := customWriter.Send()
				prometheus.Increment(prometheus.CachedResponseCounter)

				return err
			}
		} else if !requestCc.OnlyIfCached && (requestCc.MaxStaleSet || requestCc.MaxStale > -1) {
			response := stale

			if nil != response && (!modeContext.Strict || rfc.ValidateCacheControl(response, requestCc)) {
				addTime, _ := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
				rfc.SetCacheStatusHeader(response, storerName)

				responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(response.Header, "Cache-Control"))
				if responseCc.StaleWhileRevalidate > 0 {
					for h, v := range response.Header {
						customWriter.Header()[h] = v
					}
					customWriter.WriteHeader(response.StatusCode)
					rfc.HitStaleCache(&response.Header)
					customWriter.handleBuffer(func(b *bytes.Buffer) {
						_, _ = io.Copy(b, response.Body)
					})
					_, err := customWriter.Send()
					customWriter = NewCustomWriter(req, rw, bufPool)
					go func(v *core.Revalidator, goCw *CustomWriter, goRq *http.Request, goNext func(http.ResponseWriter, *http.Request) error, goCc *cacheobject.RequestCacheDirectives, goCk string, goUri string) {
						_ = s.Revalidate(v, goNext, goCw, goRq, goCc, goCk, goUri)
					}(validator, customWriter, req, next, requestCc, cachedKey, uri)
					buf := s.bufPool.Get().(*bytes.Buffer)
					buf.Reset()
					defer s.bufPool.Put(buf)

					return err
				}

				if modeContext.Bypass_response || responseCc.MustRevalidate || responseCc.NoCachePresent || validator.NeedRevalidation {
					req.Header["If-None-Match"] = append(req.Header["If-None-Match"], validator.ResponseETag)
					err := s.Revalidate(validator, next, customWriter, req, requestCc, cachedKey, uri)
					statusCode := customWriter.GetStatusCode()
					if err != nil {
						if responseCc.StaleIfError > -1 || requestCc.StaleIfError > 0 {
							code := fmt.Sprintf("; fwd-status=%d", statusCode)
							rfc.HitStaleCache(&response.Header)
							response.Header.Set("Cache-Status", response.Header.Get("Cache-Status")+code)
							maps.Copy(customWriter.Header(), response.Header)
							customWriter.WriteHeader(response.StatusCode)
							customWriter.handleBuffer(func(b *bytes.Buffer) {
								b.Reset()
								_, _ = io.Copy(b, response.Body)
							})
							_, err := customWriter.Send()

							return err
						}
						rw.WriteHeader(http.StatusGatewayTimeout)
						customWriter.handleBuffer(func(b *bytes.Buffer) {
							b.Reset()
						})
						_, err := customWriter.Send()

						return err
					}

					if statusCode == http.StatusNotModified {
						if !validator.Matched {
							rfc.SetCacheStatusHeader(response, storerName)
							customWriter.WriteHeader(response.StatusCode)
							maps.Copy(customWriter.Header(), response.Header)
							customWriter.handleBuffer(func(b *bytes.Buffer) {
								_, _ = io.Copy(b, response.Body)
							})
							_, _ = customWriter.Send()

							return err
						}
					}

					if statusCode != http.StatusNotModified && validator.Matched {
						customWriter.WriteHeader(http.StatusNotModified)
						customWriter.handleBuffer(func(b *bytes.Buffer) {
							b.Reset()
						})
						_, _ = customWriter.Send()

						return err
					}

					_, _ = customWriter.Send()

					return err
				}

				if !modeContext.Strict || rfc.ValidateMaxAgeCachedStaleResponse(requestCc, responseCc, response, int(addTime.Seconds())) != nil {
					customWriter.WriteHeader(response.StatusCode)
					rfc.HitStaleCache(&response.Header)
					maps.Copy(customWriter.Header(), response.Header)
					customWriter.handleBuffer(func(b *bytes.Buffer) {
						_, _ = io.Copy(b, response.Body)
					})
					_, err := customWriter.Send()

					return err
				}
			}
		} else if stale != nil {
			response := stale

			if !modeContext.Strict {
				rfc.SetCacheStatusHeader(response, storerName)
				customWriter.WriteHeader(response.StatusCode)
				rfc.HitStaleCache(&response.Header)
				maps.Copy(customWriter.Header(), response.Header)
				customWriter.handleBuffer(func(b *bytes.Buffer) {
					_, _ = io.Copy(b, response.Body)
				})
				_, err := customWriter.Send()

				return err
			}

			addTime, _ := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
			responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(response.Header, "Cache-Control"))

			if !modeContext.Strict || rfc.ValidateMaxAgeCachedStaleResponse(requestCc, responseCc, response, int(addTime.Seconds())) != nil {
				_, _ = time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
				rfc.SetCacheStatusHeader(response, storerName)

				responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(response.Header, "Cache-Control"))

				if responseCc.StaleIfError > -1 || requestCc.StaleIfError > 0 {
					err := s.Revalidate(validator, next, customWriter, req, requestCc, cachedKey, uri)
					statusCode := customWriter.GetStatusCode()
					if err != nil {
						code := fmt.Sprintf("; fwd-status=%d", statusCode)
						rfc.HitStaleCache(&response.Header)
						response.Header.Set("Cache-Status", response.Header.Get("Cache-Status")+code)
						maps.Copy(customWriter.Header(), response.Header)
						customWriter.WriteHeader(response.StatusCode)
						customWriter.handleBuffer(func(b *bytes.Buffer) {
							b.Reset()
							_, _ = io.Copy(b, response.Body)
						})
						_, err := customWriter.Send()

						return err
					}
				}

			}
		}
	}

	errorCacheCh := make(chan error)
	defer close(errorCacheCh)

	go func(vr *http.Request, cw *CustomWriter) {
		defer func() {
			if r := recover(); r != nil {
				s.Configuration.GetLogger().Infof("recovered due to closed errorCacheCh chan, the request context has finished prematurely %s", req.URL)
			}
		}()
		prometheus.Increment(prometheus.NoCachedResponseCounter)
		errorCacheCh <- s.Upstream(cw, vr, next, requestCc, cachedKey, uri, false)
	}(req, customWriter)

	select {
	case <-req.Context().Done():
		switch req.Context().Err() {
		case baseCtx.DeadlineExceeded:
			rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=DEADLINE-EXCEEDED")
			customWriter.Rw.WriteHeader(http.StatusGatewayTimeout)
			_, _ = customWriter.Rw.Write([]byte("Internal server error"))
			s.Configuration.GetLogger().Infof("Internal server error on endpoint %s: %v", req.URL, s.Storers)
			return baseCtx.DeadlineExceeded
		case baseCtx.Canceled:
			return baseCtx.Canceled
		default:
			return nil
		}
	case v := <-errorCacheCh:
		switch v {
		case nil:
			_, _ = customWriter.Send()
		case Upstream50xError:
			_, _ = customWriter.Send()
			return nil
		}
		return v
	}
}
