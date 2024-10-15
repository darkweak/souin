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
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate"
	"github.com/darkweak/souin/pkg/surrogate/providers"
	"github.com/pquerna/cachecontrol/cacheobject"
)

func NewHTTPCacheHandler(c configurationtypes.AbstractConfigurationInterface) *SouinBaseHandler {
	storers, err := storage.NewStorages(c)
	if err != nil {
		panic(err)
	}
	fmt.Println("Storers initialized.")
	regexpUrls := helpers.InitializeRegexp(c)
	surrogateStorage := surrogate.InitializeSurrogate(c, storers[0].Name())
	fmt.Println("Surrogate storage initialized.")
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
	fmt.Println("Souin configuration is now loaded.")

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
	Storers                  []types.Storer
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
	if !isCacheableCode(customWriter.statusCode) {
		customWriter.Headers.Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=UNCACHEABLE-STATUS-CODE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

		switch customWriter.statusCode {
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
	if responseCc == nil {
		customWriter.Headers.Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=INVALID-RESPONSE-CACHE-CONTROL", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
		return nil
	}

	modeContext := rq.Context().Value(context.Mode).(*context.ModeContext)
	if !modeContext.Bypass_request && (responseCc.PrivatePresent || rq.Header.Get("Authorization") != "") && !canBypassAuthorizationRestriction(customWriter.Header(), rq.Context().Value(context.IgnoredHeaders).([]string)) {
		customWriter.Headers.Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=PRIVATE-OR-AUTHENTICATED-RESPONSE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
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
	if ma > currentMatchedURL.TTL.Duration {
		ma = currentMatchedURL.TTL.Duration
	}

	now := rq.Context().Value(context.Now).(time.Time)
	date, _ := http.ParseTime(now.Format(http.TimeFormat))
	customWriter.Headers.Set(rfc.StoredTTLHeader, ma.String())
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
		headers := customWriter.Headers.Clone()
		for hname, shouldDelete := range responseCc.NoCache {
			if shouldDelete {
				headers.Del(hname)
			}
		}
		res := http.Response{
			StatusCode: customWriter.statusCode,
			Body:       io.NopCloser(bytes.NewBuffer(customWriter.Buf.Bytes())),
			Header:     headers,
		}

		if res.Header.Get("Date") == "" {
			res.Header.Set("Date", now.Format(http.TimeFormat))
		}
		if res.Header.Get("Content-Length") == "" {
			res.Header.Set("Content-Length", fmt.Sprint(customWriter.Buf.Len()))
		}
		res.Header.Set(rfc.StoredLengthHeader, res.Header.Get("Content-Length"))
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
						go func(currentStorer types.Storer) {
							defer wg.Done()
							if currentStorer.Set(cachedKey, response, ma) != nil {
								mu.Lock()
								fails = append(fails, fmt.Sprintf("; detail=%s-INSERTION-ERROR", currentStorer.Name()))
								mu.Unlock()
							}
						}(storer)
					}

					wg.Wait()
					if len(fails) < s.storersLen {
						go func(rs http.Response, key string) {
							_ = s.SurrogateKeyStorer.Store(&rs, key, "")
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

func (s *SouinBaseHandler) Upstream(
	customWriter *CustomWriter,
	rq *http.Request,
	next handlerFunc,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
) error {
	if err := next(customWriter, rq); err != nil {
		customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=SERVE-HTTP-ERROR", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))
		return err
	}

	s.SurrogateKeyStorer.Invalidate(rq.Method, customWriter.Header())
	if !isCacheableCode(customWriter.statusCode) {
		customWriter.Headers.Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=UNCACHEABLE-STATUS-CODE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

		switch customWriter.statusCode {
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

	select {
	case <-rq.Context().Done():
		return baseCtx.Canceled
	default:
		return s.Store(customWriter, rq, requestCc, cachedKey)
	}
}

func (s *SouinBaseHandler) Revalidate(validator *rfc.Revalidator, next handlerFunc, customWriter *CustomWriter, rq *http.Request, requestCc *cacheobject.RequestCacheDirectives, cachedKey string) error {
	err := next(customWriter, rq)
	s.SurrogateKeyStorer.Invalidate(rq.Method, customWriter.Header())

	if err == nil {
		if validator.IfUnmodifiedSincePresent && customWriter.statusCode != http.StatusNotModified {
			customWriter.Buf.Reset()
			for h, v := range customWriter.Headers {
				if len(v) > 0 {
					customWriter.Rw.Header().Set(h, strings.Join(v, ", "))
				}
			}
			customWriter.Rw.WriteHeader(http.StatusPreconditionFailed)

			return errors.New("")
		}

		if customWriter.statusCode != http.StatusNotModified {
			err = s.Store(customWriter, rq, requestCc, cachedKey)
		}
	}

	customWriter.Header().Set(
		"Cache-Status",
		fmt.Sprintf(
			"%s; fwd=request; fwd-status=%d; key=%s; detail=REQUEST-REVALIDATION",
			rq.Context().Value(context.CacheName),
			customWriter.statusCode,
			rfc.GetCacheKeyFromCtx(rq.Context()),
		),
	)
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

	// Because Yægi interpretation sucks, we have to return the empty function instead of nil.
	return false, func(w http.ResponseWriter, r *http.Request) {}
}

type handlerFunc = func(http.ResponseWriter, *http.Request) error

func (s *SouinBaseHandler) ServeHTTP(rw http.ResponseWriter, rq *http.Request, next handlerFunc) error {
	b, handler := s.HandleInternally(rq)
	if b {
		handler(rw, rq)
		return nil
	}

	rq = s.context.SetBaseContext(rq)
	cacheName := rq.Context().Value(context.CacheName).(string)
	if rq.Header.Get("Upgrade") == "websocket" || (s.ExcludeRegex != nil && s.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=EXCLUDED-REQUEST-URI")
		return next(rw, rq)
	}

	if !rq.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=UNSUPPORTED-METHOD")

		err := next(rw, rq)
		s.SurrogateKeyStorer.Invalidate(rq.Method, rw.Header())

		return err
	}

	requestCc, coErr := cacheobject.ParseRequestCacheControl(rq.Header.Get("Cache-Control"))

	modeContext := rq.Context().Value(context.Mode).(*context.ModeContext)
	if !modeContext.Bypass_request && (coErr != nil || requestCc == nil) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		err := next(rw, rq)
		s.SurrogateKeyStorer.Invalidate(rq.Method, rw.Header())

		return err
	}

	rq = s.context.SetContext(rq)

	// Yaegi sucks again, it considers false as true
	isMutationRequest := rq.Context().Value(context.IsMutationRequest).(bool)
	if isMutationRequest {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=IS-MUTATION-REQUEST")

		err := next(rw, rq)
		s.SurrogateKeyStorer.Invalidate(rq.Method, rw.Header())

		return err
	}
	cachedKey := rq.Context().Value(context.Key).(string)

	bufPool := s.bufPool.Get().(*bytes.Buffer)
	bufPool.Reset()
	defer s.bufPool.Put(bufPool)
	customWriter := NewCustomWriter(rq, rw, bufPool)
	go func(req *http.Request, crw *CustomWriter) {
		<-req.Context().Done()
		crw.mutex.Lock()
		crw.headersSent = true
		crw.mutex.Unlock()
	}(rq, customWriter)
	if modeContext.Bypass_request || !requestCc.NoCache {
		validator := rfc.ParseRequest(rq)
		var response *http.Response
		for _, currentStorer := range s.Storers {
			response = currentStorer.Prefix(cachedKey, rq, validator)
			if response != nil {
				break
			}
		}

		if response != nil && (!modeContext.Strict || rfc.ValidateCacheControl(response, requestCc)) {
			if validator.ResponseETag != "" && validator.Matched {
				rfc.SetCacheStatusHeader(response, "DEFAULT")
				customWriter.Headers = response.Header
				if validator.NotModified {
					customWriter.statusCode = http.StatusNotModified
					customWriter.Buf.Reset()
					_, _ = customWriter.Send()

					return nil
				}

				customWriter.statusCode = response.StatusCode
				_, _ = io.Copy(customWriter.Buf, response.Body)
				_, _ = customWriter.Send()

				return nil
			}

			if validator.NeedRevalidation {
				err := s.Revalidate(validator, next, customWriter, rq, requestCc, cachedKey)
				_, _ = customWriter.Send()

				return err
			}
			if resCc, _ := cacheobject.ParseResponseCacheControl(response.Header.Get("Cache-Control")); resCc.NoCachePresent {
				err := s.Revalidate(validator, next, customWriter, rq, requestCc, cachedKey)
				_, _ = customWriter.Send()

				return err
			}
			rfc.SetCacheStatusHeader(response, "DEFAULT")
			if !modeContext.Strict || rfc.ValidateMaxAgeCachedResponse(requestCc, response) != nil {
				customWriter.Headers = response.Header
				customWriter.statusCode = response.StatusCode
				_, _ = io.Copy(customWriter.Buf, response.Body)
				_, err := customWriter.Send()

				return err
			}
		} else if response == nil && !requestCc.OnlyIfCached && (requestCc.MaxStaleSet || requestCc.MaxStale > -1) {
			for _, currentStorer := range s.Storers {
				response = currentStorer.Prefix(storage.StalePrefix+cachedKey, rq, validator)
				if response != nil {
					break
				}
			}
			if nil != response && (!modeContext.Strict || rfc.ValidateCacheControl(response, requestCc)) {
				addTime, _ := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
				rfc.SetCacheStatusHeader(response, "DEFAULT")

				responseCc, _ := cacheobject.ParseResponseCacheControl(response.Header.Get("Cache-Control"))
				if responseCc.StaleWhileRevalidate > 0 {
					customWriter.Headers = response.Header
					customWriter.statusCode = response.StatusCode
					rfc.HitStaleCache(&response.Header)
					_, _ = io.Copy(customWriter.Buf, response.Body)
					_, err := customWriter.Send()
					customWriter = NewCustomWriter(rq, rw, bufPool)
					go func(v *rfc.Revalidator, goCw *CustomWriter, goRq *http.Request, goNext func(http.ResponseWriter, *http.Request) error, goCc *cacheobject.RequestCacheDirectives, goCk string) {
						_ = s.Revalidate(v, goNext, goCw, goRq, goCc, goCk)
					}(validator, customWriter, rq, next, requestCc, cachedKey)
					buf := s.bufPool.Get().(*bytes.Buffer)
					buf.Reset()
					defer s.bufPool.Put(buf)

					return err
				}

				if responseCc.MustRevalidate || responseCc.NoCachePresent || validator.NeedRevalidation {
					rq.Header["If-None-Match"] = append(rq.Header["If-None-Match"], validator.ResponseETag)
					err := s.Revalidate(validator, next, customWriter, rq, requestCc, cachedKey)
					if err != nil {
						if responseCc.StaleIfError > -1 || requestCc.StaleIfError > 0 {
							code := fmt.Sprintf("; fwd-status=%d", customWriter.statusCode)
							customWriter.Headers = response.Header
							customWriter.statusCode = response.StatusCode
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

					if customWriter.statusCode == http.StatusNotModified {
						if !validator.Matched {
							rfc.SetCacheStatusHeader(response, "DEFAULT")
							customWriter.statusCode = response.StatusCode
							customWriter.Headers = response.Header
							_, _ = io.Copy(customWriter.Buf, response.Body)
							_, _ = customWriter.Send()

							return err
						}
					}

					if customWriter.statusCode != http.StatusNotModified && validator.Matched {
						customWriter.statusCode = http.StatusNotModified
						customWriter.Buf.Reset()
						_, _ = customWriter.Send()

						return err
					}

					_, _ = customWriter.Send()

					return err
				}

				if !modeContext.Strict || rfc.ValidateMaxAgeCachedStaleResponse(requestCc, response, int(addTime.Seconds())) != nil {
					customWriter.Headers = response.Header
					customWriter.statusCode = response.StatusCode
					rfc.HitStaleCache(&response.Header)
					_, _ = io.Copy(customWriter.Buf, response.Body)
					_, err := customWriter.Send()

					return err
				}
			}
		}
	}

	errorCacheCh := make(chan error)
	go func() {
		errorCacheCh <- s.Upstream(customWriter, rq, next, requestCc, cachedKey)
	}()

	select {
	case <-rq.Context().Done():
		switch rq.Context().Err() {
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
