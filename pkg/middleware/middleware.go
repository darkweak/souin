package middleware

import (
	"bytes"
	baseCtx "context"
	"errors"
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
	"github.com/darkweak/souin/pkg/api/prometheus"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/surrogate"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/pquerna/cachecontrol/cacheobject"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/singleflight"
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
		singleflightPool:         singleflight.Group{},
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
	singleflightPool         singleflight.Group
	bufPool                  *sync.Pool
	storersLen               int
}

type upsreamError struct{}

func (upsreamError) Error() string {
	return "Upstream error"
}

func isCacheableCode(code int) bool {
	switch code {
	case 200, 203, 204, 206, 300, 301, 404, 405, 410, 414, 501:
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

func (s *SouinBaseHandler) Store(
	customWriter *CustomWriter,
	rq *http.Request,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
) error {
	statusCode := customWriter.GetStatusCode()
	if !isCacheableCode(statusCode) {
		customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=UNCACHEABLE-STATUS-CODE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

		switch statusCode {
		case 500, 502, 503, 504:
			return new(upsreamError)
		}

		return nil
	}

	if customWriter.Header().Get("Cache-Control") == "" {
		// TODO see with @mnot if mandatory to not store the response when no Cache-Control given.
		// if s.DefaultMatchedUrl.DefaultCacheControl == "" {
		// 	customWriter.Headers.Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=EMPTY-RESPONSE-CACHE-CONTROL", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
		// 	return nil
		// }
		customWriter.Header().Set("Cache-Control", s.DefaultMatchedUrl.DefaultCacheControl)
	}

	responseCc, _ := cacheobject.ParseResponseCacheControl(customWriter.Header().Get("Cache-Control"))
	s.Configuration.GetLogger().Sugar().Debugf("Response cache-control %+v", responseCc)
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

	ma := currentMatchedURL.TTL.Duration
	if responseCc.SMaxAge >= 0 {
		ma = time.Duration(responseCc.SMaxAge) * time.Second
	} else if responseCc.MaxAge >= 0 {
		ma = time.Duration(responseCc.MaxAge) * time.Second
	}

	now := rq.Context().Value(context.Now).(time.Time)
	date, _ := http.ParseTime(now.Format(http.TimeFormat))
	customWriter.Header().Set(rfc.StoredTTLHeader, ma.String())
	ma = ma - time.Since(date)

	if exp := customWriter.Header().Get("Expires"); exp != "" {
		delta, _ := time.Parse(exp, time.RFC1123)
		if sub := delta.Sub(now); sub > 0 {
			ma = sub
		}
	}

	status := fmt.Sprintf("%s; fwd=uri-miss", rq.Context().Value(context.CacheName))
	if (modeContext.Bypass_request || !requestCc.NoStore) &&
		(modeContext.Bypass_response || !responseCc.NoStore) {
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
		res.Header.Set(rfc.StoredLengthHeader, res.Header.Get("Content-Length"))
		response, err := httputil.DumpResponse(&res, true)
		if err == nil && bLen > 0 {
			variedHeaders, isVaryStar := rfc.VariedHeaderAllCommaSepValues(res.Header)
			if isVaryStar {
				// "Implies that the response is uncacheable"
				status += "; detail=UPSTREAM-VARY-STAR"
			} else {
				cachedKey += rfc.GetVariedCacheKey(rq, variedHeaders)
				s.Configuration.GetLogger().Sugar().Debugf("Store the response %+v with duration %v", res, ma)

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
							if currentStorer.Set(cachedKey, response, currentMatchedURL, ma) == nil {
								s.Configuration.GetLogger().Sugar().Debugf("Stored the key %s in the %s provider", cachedKey, currentStorer.Name())
							} else {
								mu.Lock()
								fails = append(fails, fmt.Sprintf("; detail=%s-INSERTION-ERROR", currentStorer.Name()))
								mu.Unlock()
							}
						}(storer)
					}

					wg.Wait()
					if len(fails) < s.storersLen {
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
	body           []byte
	headers        http.Header
	requestHeaders http.Header
	code           int
}

func (s *SouinBaseHandler) Upstream(
	customWriter *CustomWriter,
	rq *http.Request,
	next handlerFunc,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
) error {
	s.Configuration.GetLogger().Sugar().Debug("Request the upstream server")
	prometheus.Increment(prometheus.RequestCounter)

	sfValue, err, shared := s.singleflightPool.Do(cachedKey, func() (interface{}, error) {
		if e := next(customWriter, rq); e != nil {
			s.Configuration.GetLogger().Sugar().Warnf("%#v", e)
			customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=SERVE-HTTP-ERROR", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
			return nil, e
		}

		s.SurrogateKeyStorer.Invalidate(rq.Method, customWriter.Header())

		statusCode := customWriter.GetStatusCode()
		if !isCacheableCode(statusCode) {
			customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=UNCACHEABLE-STATUS-CODE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

			switch statusCode {
			case 500, 502, 503, 504:
				return nil, new(upsreamError)
			}
		}

		if customWriter.Header().Get("Cache-Control") == "" {
			customWriter.Header().Set("Cache-Control", s.DefaultMatchedUrl.DefaultCacheControl)
		}

		err := s.Store(customWriter, rq, requestCc, cachedKey)
		defer customWriter.Buf.Reset()
		return singleflightValue{
			body:           customWriter.Buf.Bytes(),
			headers:        customWriter.Header().Clone(),
			requestHeaders: rq.Header,
			code:           statusCode,
		}, err
	})

	if err != nil {
		return err
	}

	if sfWriter, ok := sfValue.(singleflightValue); ok && shared {
		if vary := sfWriter.headers.Get("Vary"); vary != "" {
			variedHeaders, isVaryStar := rfc.VariedHeaderAllCommaSepValues(sfWriter.headers)
			if !isVaryStar {
				for _, vh := range variedHeaders {
					if rq.Header.Get(vh) != sfWriter.requestHeaders.Get(vh) {
						cachedKey += rfc.GetVariedCacheKey(rq, variedHeaders)
						return s.Upstream(customWriter, rq, next, requestCc, cachedKey)
					}
				}
			}
		}
		s.Configuration.GetLogger().Sugar().Infof("Reused response from concurrent request with the key %s", cachedKey)
		customWriter.Write(sfWriter.body)
		for h, v := range sfWriter.headers {
			customWriter.Header()[h] = v
		}
		customWriter.WriteHeader(sfWriter.code)
	}

	return nil
}

func (s *SouinBaseHandler) Revalidate(validator *rfc.Revalidator, next handlerFunc, customWriter *CustomWriter, rq *http.Request, requestCc *cacheobject.RequestCacheDirectives, cachedKey string) error {
	s.Configuration.GetLogger().Sugar().Debug("Revalidate the request with the upstream server")
	prometheus.Increment(prometheus.RequestRevalidationCounter)

	sfValue, err, shared := s.singleflightPool.Do(cachedKey, func() (interface{}, error) {
		err := next(customWriter, rq)
		s.SurrogateKeyStorer.Invalidate(rq.Method, customWriter.Header())

		statusCode := customWriter.GetStatusCode()
		if err == nil {
			if validator.IfUnmodifiedSincePresent && statusCode != http.StatusNotModified {
				customWriter.Buf.Reset()
				customWriter.Rw.WriteHeader(http.StatusPreconditionFailed)

				return nil, errors.New("")
			}

			if statusCode != http.StatusNotModified {
				err = s.Store(customWriter, rq, requestCc, cachedKey)
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

		defer customWriter.Buf.Reset()
		return singleflightValue{
			body:    customWriter.Buf.Bytes(),
			headers: customWriter.Header().Clone(),
			code:    statusCode,
		}, err
	})

	if err != nil {
		return err
	}
	if sfWriter, ok := sfValue.(singleflightValue); ok && shared {
		s.Configuration.GetLogger().Sugar().Infof("Reused response from concurrent request with the key %s", cachedKey)
		customWriter.Write(sfWriter.body)
		for h, v := range sfWriter.headers {
			customWriter.Header()[h] = v
		}
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

func (s *SouinBaseHandler) ServeHTTP(rw http.ResponseWriter, rq *http.Request, next handlerFunc) error {
	start := time.Now()
	defer func(s time.Time) {
		prometheus.Add(prometheus.AvgResponseTime, float64(time.Since(s).Milliseconds()))
	}(start)
	s.Configuration.GetLogger().Sugar().Debugf("Incomming request %+v", rq)
	if b, handler := s.HandleInternally(rq); b {
		handler(rw, rq)
		return nil
	}

	req := s.context.SetBaseContext(rq)
	cacheName := req.Context().Value(context.CacheName).(string)
	if rq.Header.Get("Upgrade") == "websocket" || (s.ExcludeRegex != nil && s.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=EXCLUDED-REQUEST-URI")
		return next(rw, req)
	}

	if !req.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=UNSUPPORTED-METHOD")

		err := next(rw, req)
		s.SurrogateKeyStorer.Invalidate(req.Method, rw.Header())

		return err
	}

	requestCc, coErr := cacheobject.ParseRequestCacheControl(req.Header.Get("Cache-Control"))

	modeContext := req.Context().Value(context.Mode).(*context.ModeContext)
	if !modeContext.Bypass_request && (coErr != nil || requestCc == nil) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		err := next(rw, req)
		s.SurrogateKeyStorer.Invalidate(req.Method, rw.Header())

		return err
	}

	req = s.context.SetContext(req, rq)
	if req.Context().Value(context.IsMutationRequest).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=IS-MUTATION-REQUEST")

		err := next(rw, req)
		s.SurrogateKeyStorer.Invalidate(req.Method, rw.Header())

		return err
	}
	cachedKey := req.Context().Value(context.Key).(string)

	bufPool := s.bufPool.Get().(*bytes.Buffer)
	bufPool.Reset()
	defer s.bufPool.Put(bufPool)
	customWriter := NewCustomWriter(req, rw, bufPool)
	go func(req *http.Request, crw *CustomWriter) {
		<-req.Context().Done()
		crw.mutex.Lock()
		crw.headersSent = true
		crw.mutex.Unlock()
	}(req, customWriter)
	s.Configuration.GetLogger().Sugar().Debugf("Request cache-control %+v", requestCc)
	if modeContext.Bypass_request || !requestCc.NoCache {
		validator := rfc.ParseRequest(req)
		var response *http.Response
		for _, currentStorer := range s.Storers {
			response = currentStorer.Prefix(cachedKey, req, validator)
			if response != nil {
				s.Configuration.GetLogger().Sugar().Debugf("Found response in the %s storage", currentStorer.Name())
				break
			}
		}

		if response != nil && (!modeContext.Strict || rfc.ValidateCacheControl(response, requestCc)) {
			if validator.ResponseETag != "" && validator.Matched {
				rfc.SetCacheStatusHeader(response)
				for h, v := range response.Header {
					customWriter.Header()[h] = v
				}
				if validator.NotModified {
					customWriter.WriteHeader(http.StatusNotModified)
					customWriter.Buf.Reset()
					_, _ = customWriter.Send()

					return nil
				}

				customWriter.WriteHeader(response.StatusCode)
				_, _ = io.Copy(customWriter.Buf, response.Body)
				_, _ = customWriter.Send()

				return nil
			}

			if validator.NeedRevalidation {
				err := s.Revalidate(validator, next, customWriter, req, requestCc, cachedKey)
				_, _ = customWriter.Send()

				return err
			}
			if resCc, _ := cacheobject.ParseResponseCacheControl(response.Header.Get("Cache-Control")); resCc.NoCachePresent {
				prometheus.Increment(prometheus.NoCachedResponseCounter)
				err := s.Revalidate(validator, next, customWriter, req, requestCc, cachedKey)
				_, _ = customWriter.Send()

				return err
			}
			rfc.SetCacheStatusHeader(response)
			if !modeContext.Strict || rfc.ValidateMaxAgeCachedResponse(requestCc, response) != nil {
				for h, v := range response.Header {
					customWriter.Header()[h] = v
				}
				customWriter.WriteHeader(response.StatusCode)
				s.Configuration.GetLogger().Sugar().Debugf("Serve from cache %+v", req)
				_, _ = io.Copy(customWriter.Buf, response.Body)
				_, err := customWriter.Send()
				prometheus.Increment(prometheus.CachedResponseCounter)

				return err
			}
		} else if response == nil && !requestCc.OnlyIfCached && (requestCc.MaxStaleSet || requestCc.MaxStale > -1) {
			for _, currentStorer := range s.Storers {
				response = currentStorer.Prefix(storage.StalePrefix+cachedKey, req, validator)
				if response != nil {
					break
				}
			}
			if nil != response && (!modeContext.Strict || rfc.ValidateCacheControl(response, requestCc)) {
				addTime, _ := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
				rfc.SetCacheStatusHeader(response)

				responseCc, _ := cacheobject.ParseResponseCacheControl(response.Header.Get("Cache-Control"))
				if responseCc.StaleWhileRevalidate > 0 {
					for h, v := range response.Header {
						customWriter.Header()[h] = v
					}
					customWriter.WriteHeader(response.StatusCode)
					rfc.HitStaleCache(&response.Header)
					_, _ = io.Copy(customWriter.Buf, response.Body)
					_, err := customWriter.Send()
					customWriter = NewCustomWriter(req, rw, bufPool)
					go func(v *rfc.Revalidator, goCw *CustomWriter, goRq *http.Request, goNext func(http.ResponseWriter, *http.Request) error, goCc *cacheobject.RequestCacheDirectives, goCk string) {
						_ = s.Revalidate(v, goNext, goCw, goRq, goCc, goCk)
					}(validator, customWriter, req, next, requestCc, cachedKey)
					buf := s.bufPool.Get().(*bytes.Buffer)
					buf.Reset()
					defer s.bufPool.Put(buf)

					return err
				}

				if responseCc.MustRevalidate || responseCc.NoCachePresent || validator.NeedRevalidation {
					req.Header["If-None-Match"] = append(req.Header["If-None-Match"], validator.ResponseETag)
					err := s.Revalidate(validator, next, customWriter, req, requestCc, cachedKey)
					statusCode := customWriter.GetStatusCode()
					if err != nil {
						if responseCc.StaleIfError > -1 || requestCc.StaleIfError > 0 {
							code := fmt.Sprintf("; fwd-status=%d", statusCode)
							for h, v := range response.Header {
								customWriter.Header()[h] = v
							}
							customWriter.WriteHeader(response.StatusCode)
							rfc.HitStaleCache(&response.Header)
							response.Header.Set("Cache-Status", response.Header.Get("Cache-Status")+code)
							_, _ = io.Copy(customWriter.Buf, response.Body)
							_, err := customWriter.Send()

							return err
						}
						rw.WriteHeader(http.StatusGatewayTimeout)
						customWriter.Buf.Reset()
						_, err := customWriter.Send()

						return err
					}

					if statusCode == http.StatusNotModified {
						if !validator.Matched {
							rfc.SetCacheStatusHeader(response)
							statusCode = response.StatusCode
							for h, v := range response.Header {
								customWriter.Header()[h] = v
							}
							_, _ = io.Copy(customWriter.Buf, response.Body)
							_, _ = customWriter.Send()

							return err
						}
					}

					if statusCode != http.StatusNotModified && validator.Matched {
						customWriter.WriteHeader(http.StatusNotModified)
						customWriter.Buf.Reset()
						_, _ = customWriter.Send()

						return err
					}

					_, _ = customWriter.Send()

					return err
				}

				if rfc.ValidateMaxAgeCachedStaleResponse(requestCc, response, int(addTime.Seconds())) != nil {
					for h, v := range response.Header {
						customWriter.Header()[h] = v
					}
					customWriter.WriteHeader(response.StatusCode)
					rfc.HitStaleCache(&response.Header)
					_, _ = io.Copy(customWriter.Buf, response.Body)
					_, err := customWriter.Send()

					return err
				}
			}
		}
	} else {
		prometheus.Increment(prometheus.NoCachedResponseCounter)
	}

	errorCacheCh := make(chan error)
	go func(vr *http.Request, cw *CustomWriter) {
		errorCacheCh <- s.Upstream(cw, vr, next, requestCc, cachedKey)
	}(req, customWriter)

	select {
	case <-req.Context().Done():
		switch req.Context().Err() {
		case baseCtx.DeadlineExceeded:
			customWriter.WriteHeader(http.StatusGatewayTimeout)
			rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=DEADLINE-EXCEEDED")
			_, _ = customWriter.Rw.Write([]byte("Internal server error"))
			return baseCtx.DeadlineExceeded
		case baseCtx.Canceled:
			return baseCtx.Canceled
		default:
			return nil
		}
	case v := <-errorCacheCh:
		if v == nil {
			_, _ = customWriter.Send()
		}
		return v
	}
}
