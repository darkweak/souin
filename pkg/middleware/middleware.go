package middleware

import (
	"bytes"
	baseCtx "context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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
					// Extend the expiry so walks longer than the TTL keep ownership.
					renewed := evictionLockHolder + "|" + now.Add(evictionLockTTL).Format(time.RFC3339)
					_ = storer.Set(evictionLockKey, []byte(renewed), evictionLockTTL)

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

// evictionRegistry tracks storers that already have an eviction goroutine so
// re-instantiated handlers (e.g. on config reload or multiple cache blocks)
// don't spawn concurrent walkers for the same storer.
var evictionRegistry = sync.Map{}

func registerMappingKeysEviction(ctx baseCtx.Context, logger core.Logger, storers []types.Storer, interval time.Duration) {
	for _, storer := range storers {
		if _, alreadyRegistered := evictionRegistry.LoadOrStore(storer.Name()+"-"+storer.Uuid(), true); alreadyRegistered {
			logger.Debugf("mapping eviction already registered for storer %s", storer.Name())

			continue
		}

		logger.Debugf("registering mapping eviction for storer %s (interval: %s)", storer.Name(), interval)
		go func(current types.Storer, currentInterval time.Duration) {
			// A timer reset after each walk guarantees a full interval of
			// rest between walks; a ticker would fire immediately after a
			// walk longer than the interval.
			timer := time.NewTimer(currentInterval)
			defer timer.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					if tryAcquireEvictionLock(current) {
						// Keep extending the lock while the walk runs so other
						// replicas don't join once the TTL elapses mid-walk.
						walkDone := make(chan struct{})
						go func() {
							keepalive := time.NewTicker(evictionLockTTL / 2)
							defer keepalive.Stop()

							for {
								select {
								case <-walkDone:
									return
								case <-keepalive.C:
									_ = tryAcquireEvictionLock(current)
								}
							}
						}()

						logger.Debugf("run mapping eviction for storer %s", current.Name())
						api.EvictMapping(current)
						close(walkDone)
					} else {
						logger.Debugf("skipping mapping eviction for storer %s, another instance holds the lock", current.Name())
					}

					timer.Reset(currentInterval)
				}
			}
		}(storer, interval)
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
	c.GetLogger().Debugf("Configuration: %#v.", c.GetDefaultCache())

	evictionCtx, _ := signal.NotifyContext(baseCtx.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	registerMappingKeysEviction(evictionCtx, c.GetLogger(), storers, c.GetDefaultCache().GetMappingEvictionInterval())

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

// A representation that has not changed for a while is assumed to stay valid
// for a tenth of that time, and never for more than a day. RFC 9111 section
// 4.2.2 leaves the exact heuristic to the cache but warns against long
// lifetimes.
const (
	heuristicFreshnessRatio = 10
	heuristicFreshnessCap   = 24 * time.Hour
)

// heuristicFreshness derives a freshness lifetime from the `Last-Modified` of
// a response that carries no explicit expiry, and reports whether it could.
func heuristicFreshness(headers http.Header, statusCode int, responseCc *cacheobject.ResponseCacheDirectives) (time.Duration, bool) {
	if !isCacheableCode(statusCode) && !responseCc.Public {
		return 0, false
	}

	lastModified, err := rfc.ParseHTTPDate(headers.Get("Last-Modified"))
	if err != nil {
		return 0, false
	}

	date, err := rfc.ParseHTTPDate(headers.Get("Date"))
	if err != nil {
		return 0, false
	}

	delta := date.Sub(lastModified)
	if delta <= 0 {
		return 0, false
	}

	lifetime := delta / heuristicFreshnessRatio
	if lifetime > heuristicFreshnessCap {
		lifetime = heuristicFreshnessCap
	}

	return lifetime, true
}

// isStorableCode tells whether a response may be stored at all. RFC 9111
// section 3 lets a cache keep any final status code, as long as the response
// says for how long it stays fresh; the hardcoded list only covers the codes
// that are cacheable by default, without the origin saying anything.
func (s *SouinBaseHandler) isStorableCode(code int, headers http.Header) bool {
	if isCacheableCode(code) || s.hasAllowedAdditionalStatusCodesToCache(code) {
		return true
	}

	if code < 200 || code > 599 {
		return false
	}

	if headers.Get("Expires") != "" {
		return true
	}

	headerName, cacheControl := s.SurrogateKeyStorer.GetSurrogateControl(headers)
	if cacheControl == "" {
		return false
	}

	responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(headers, headerName))

	return responseCc != nil && (responseCc.MaxAge >= 0 || responseCc.SMaxAge >= 0 || responseCc.Public)
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

func dumpResponse(statusCode int, headers http.Header, body []byte) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(256 + len(body))

	_, _ = buf.WriteString("HTTP/1.1 ")
	_, _ = buf.WriteString(strconv.Itoa(statusCode))
	_, _ = buf.WriteString(" ")
	_, _ = buf.WriteString(http.StatusText(statusCode))
	_, _ = buf.WriteString("\r\n")

	err := headers.Write(&buf)
	if err != nil {
		return nil, fmt.Errorf("cannot write headers to the buffer: %w", err)
	}

	_, _ = buf.WriteString("\r\n")

	_, err = buf.Write(body)
	if err != nil {
		return nil, fmt.Errorf("cannot write headers to the buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// hopByHopHeaders are the connection-specific header fields an intermediary
// must not forward (RFC 9110 §7.6.1). Stripping them before storage keeps one
// connection's framing/handshake from being replayed to later clients. This is
// a message transformation that does not change the content, so it is allowed
// even when the response carries Cache-Control: no-transform (RFC 9110 §7.7).
var hopByHopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// removeHopByHopHeaders deletes the connection-specific headers from h,
// including any field-names listed in the Connection header itself.
func removeHopByHopHeaders(h http.Header) {
	for _, name := range rfc.HeaderAllCommaSepValues(h, "Connection") {
		h.Del(name)
	}
	for _, name := range hopByHopHeaders {
		h.Del(name)
	}
}

// revalidationRetention is how long a stale response is kept on top of any
// explicitly configured stale window. Without it the cache would have nothing
// left to revalidate the moment a response turns stale, and would have to
// refetch a body the origin could have answered with a bare `304`.
const revalidationRetention = 5 * time.Minute

// retentionWindow returns for how long a response is kept once it turned
// stale. The storer adds the configured stale window on its own, so what is
// left to cover here is the room a `stale-while-revalidate` may need and the
// room a revalidation needs.
func (s *SouinBaseHandler) retentionWindow(responseCc *cacheobject.ResponseCacheDirectives) time.Duration {
	window := revalidationRetention

	if responseCc != nil {
		if swr := time.Duration(responseCc.StaleWhileRevalidate) * time.Second; swr > window {
			window = swr
		}
	}

	return window
}

func (s *SouinBaseHandler) Store(
	customWriter *CustomWriter,
	rq *http.Request,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
) error {
	statusCode := customWriter.GetStatusCode()
	if !s.isStorableCode(statusCode, customWriter.Header()) {
		cacheName := rq.Context().Value(context.CacheName).(string)
		cacheKey := rfc.GetCacheKeyFromCtx(rq.Context())
		customWriter.Header().Set("Cache-Status", cacheName+"; fwd=uri-miss; key="+cacheKey+"; detail=UNCACHEABLE-STATUS-CODE")

		switch statusCode {
		case 500, 502, 503, 504:
			return Upstream50xError
		}

		return nil
	}

	headerName, cacheControl := s.SurrogateKeyStorer.GetSurrogateControl(customWriter.Header())
	// The configured default only stands in for a response that carries no
	// cache directives of its own. Keeping track of it lets the explicit
	// freshness information the response may still hold (`Expires`) win over
	// that default.
	usesDefaultCacheControl := cacheControl == ""
	if usesDefaultCacheControl {
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

	// RFC 9111 section 3.5: a shared cache may store the response to an
	// authenticated request when the response explicitly allows it.
	authenticatedButStorable := responseCc.MustRevalidate || responseCc.Public || responseCc.SMaxAge >= 0 ||
		canBypassAuthorizationRestriction(customWriter.Header(), rq.Context().Value(context.IgnoredHeaders).([]string))

	modeContext := rq.Context().Value(context.Mode).(*context.ModeContext)
	if !modeContext.Bypass_request && (responseCc.PrivatePresent || rq.Header.Get("Authorization") != "") && !authenticatedButStorable {
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

	now := rq.Context().Value(context.Now).(time.Time)

	ma := currentMatchedURL.TTL.Duration
	hasExplicitFreshness := false
	if !modeContext.Bypass_response {
		if responseCc.SMaxAge >= 0 {
			ma = time.Duration(responseCc.SMaxAge) * time.Second
			hasExplicitFreshness = true
		} else if responseCc.MaxAge >= 0 {
			ma = time.Duration(responseCc.MaxAge) * time.Second
			hasExplicitFreshness = true
		} else if expires := customWriter.Header().Values("Expires"); len(expires) > 0 {
			// A response carrying several `Expires` lines has no single
			// expiry to honour, so it is treated as already expired.
			exp, err := rfc.ParseHTTPDate(expires[0])
			if len(expires) > 1 {
				err = errors.New("several Expires header lines")
			}
			if err != nil {
				customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=MALFORMED-EXPIRES", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

				return nil
			}

			// RFC 9111 section 4.2.1: the freshness lifetime an `Expires`
			// header conveys is `Expires` minus `Date`, not the delay until
			// `Expires` measured on the cache clock.
			date, dateErr := rfc.ParseHTTPDate(customWriter.Header().Get("Date"))
			if dateErr != nil {
				date = now
			}

			duration := exp.Sub(date)
			if duration <= 0 {
				customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=EXPIRED-RESPONSE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

				return nil
			}

			// An `Expires` far enough in the future means "as long as you
			// like": clamp it rather than refuse to store the response.
			if duration > 10*types.OneYearDuration {
				duration = 10 * types.OneYearDuration
			}

			ma = duration
			hasExplicitFreshness = true
		}
	}

	usesHeuristicFreshness := false
	if !modeContext.Bypass_response && !hasExplicitFreshness {
		if heuristic, ok := heuristicFreshness(customWriter.Header(), statusCode, responseCc); ok {
			ma = heuristic
			usesHeuristicFreshness = true
		}
	}

	// The stored TTL is the freshness lifetime the response was given, which
	// is what the `ttl` of the Cache-Status header counts down from. The
	// instant the response turns stale is a different thing: RFC 9111
	// section 4.2 measures freshness against the response age, so the age it
	// already carried when it reached this cache eats into its lifetime.
	customWriter.Header().Set(rfc.StoredTTLHeader, ma.String())

	ma -= rfc.InitialAge(customWriter.Header(), now)
	rfc.SetStoredExpiry(customWriter.Header(), now.Add(ma))

	// A response that already reached the end of its freshness lifetime is
	// still worth storing: it can be revalidated, and the directives that
	// allow stale content to be served need something to serve. How long it
	// is kept is decided by the retention window, not by what is left of its
	// lifetime.
	storeDuration := s.retentionWindow(responseCc)
	if ma > 0 {
		storeDuration += ma
	}

	// RFC 9111 section 5.2.2.3: `must-understand` tells a cache that does
	// understand the status code to ignore an accompanying `no-store`.
	mustUnderstand := hasExplicitFreshness && isCacheableCode(statusCode) &&
		slices.Contains(rfc.HeaderAllCommaSepValues(customWriter.Header(), headerName), "must-understand")

	status := fmt.Sprintf("%s; fwd=uri-miss", rq.Context().Value(context.CacheName))

	if (modeContext.Bypass_request || !requestCc.NoStore) &&
		(modeContext.Bypass_response || !responseCc.NoStore || mustUnderstand ||
			(usesDefaultCacheControl && (hasExplicitFreshness || usesHeuristicFreshness))) {
		headers := customWriter.Header().Clone()
		for hname, shouldDelete := range responseCc.NoCache {
			if shouldDelete {
				headers.Del(hname)
			}
		}
		removeHopByHopHeaders(headers)

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
		response, err := dumpResponse(res.StatusCode, res.Header, b)
		if err == nil && (bLen > 0 || rq.Method == http.MethodHead || canStatusCodeEmptyContent(statusCode) || s.hasAllowedAdditionalStatusCodesToCache(statusCode)) {
			variedHeaders, isVaryStar := rfc.VariedHeaderAllCommaSepValues(res.Header)
			if isVaryStar {
				// "Implies that the response is uncacheable"
				status += "; detail=UPSTREAM-VARY-STAR"
			} else {
				variedKey := cachedKey + rfc.GetVariedCacheKey(rq, variedHeaders)
				if rq.Context().Value(context.Hashed).(bool) {
					cachedKey = strconv.FormatUint(xxhash.Sum64String(cachedKey), 10)
					variedKey = strconv.FormatUint(xxhash.Sum64String(variedKey), 10)
				}
				s.Configuration.GetLogger().Debugf("Store the response for %s with duration %v", variedKey, storeDuration)

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
							res.Header.Get("Etag"), storeDuration,
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
									currentRes.Header.Get("Etag"), storeDuration,
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
		if !s.isStorableCode(statusCode, customWriter.Header()) {
			customWriter.Header().Set("Cache-Status", fmt.Sprintf("%s; fwd=uri-miss; key=%s; detail=UNCACHEABLE-STATUS-CODE", rq.Context().Value(context.CacheName), rfc.GetCacheKeyFromCtx(rq.Context())))

			switch statusCode {
			case 500, 502, 503, 504:
				return nil, Upstream50xError
			}
		}

		err := s.Store(customWriter, rq, requestCc, cachedKey, uri)

		// Store applies the configured default when the response carries no
		// cache directives, so it has to run first: overwriting the header
		// here would hide from it whether the directives are the response's
		// own ones or the configured fallback.
		headerName, cacheControl := s.SurrogateKeyStorer.GetSurrogateControl(customWriter.Header())
		if cacheControl == "" {
			cacheControl = s.DefaultMatchedUrl.DefaultCacheControl
			customWriter.Header().Set(headerName, cacheControl)
		}

		// Copy the buffer bytes so the returned value is independent of the
		// underlying buffer, which may be reset or returned to the pool.
		bodySnapshot := make([]byte, customWriter.Buf.Len())
		copy(bodySnapshot, customWriter.Buf.Bytes())

		defer customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})

		return singleflightValue{
			body:              bodySnapshot,
			headers:           customWriter.Header().Clone(),
			requestHeaders:    rq.Header.Clone(),
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
		customWriter.Buf.Reset()
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

		// Copy the buffer bytes so the returned value is independent of the
		// underlying buffer, which may be reset or returned to the pool.
		bodySnapshot := make([]byte, customWriter.Buf.Len())
		copy(bodySnapshot, customWriter.Buf.Bytes())

		defer customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})
		return singleflightValue{
			body:    bodySnapshot,
			headers: customWriter.Header().Clone(),
			code:    statusCode,
		}, err
	})

	if sfWriter, ok := sfValue.(singleflightValue); ok {
		if shared {
			s.Configuration.GetLogger().Infof("Reused response from concurrent request with the key %s", cachedKey)
		}
		customWriter.Buf.Reset()
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

	expiry, ok := rfc.StoredExpiry(response.Header)
	if !ok {
		return
	}

	headerName, _ := s.SurrogateKeyStorer.GetSurrogateControl(response.Header)
	responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(response.Header, headerName))

	ma := s.retentionWindow(responseCc)
	if remaining := time.Until(expiry); remaining > 0 {
		ma += remaining
	}

	variedHeaders, _ := rfc.VariedHeaderAllCommaSepValues(response.Header)
	variedKey := cachedKey + rfc.GetVariedCacheKey(rq, variedHeaders)

	if rq.Context().Value(context.Hashed).(bool) {
		cachedKey = strconv.FormatUint(xxhash.Sum64String(cachedKey), 10)
		variedKey = strconv.FormatUint(xxhash.Sum64String(variedKey), 10)
	}

	vhs := http.Header{}
	for _, hname := range variedHeaders {
		hn := strings.Split(hname, ":")
		vhs.Set(hn[0], rq.Header.Get(hn[0]))
	}

	bodyResponse := new(bytes.Buffer)
	_, _ = io.Copy(bodyResponse, response.Body)

	_ = response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(bodyResponse.Bytes()))
	res, _ := dumpResponse(response.StatusCode, response.Header, bodyResponse.Bytes())

	for _, currentStorer := range s.Storers[:idx] {
		err := currentStorer.SetMultiLevel(
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
	s.Configuration.GetLogger().Debugf("Incoming request %+v", rq)
	if b, handler := s.HandleInternally(rq); b {
		handler(rw, rq)
		return nil
	}

	req := s.context.SetBaseContext(rq)
	defer func() {
		toCancel := req.Context().Value(context.TimeoutCancel)

		if toCancel != nil {
			toCancel.(baseCtx.CancelFunc)()
		}
	}()

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
			// RFC 9111 section 4.4: an unsafe method that succeeded
			// invalidates the target URI, and with it the `Location` and
			// `Content-Location` it points at as long as they share the
			// target's origin.
			targets := []*url.URL{req.URL}
			for _, name := range []string{"Location", "Content-Location"} {
				value := rw.Header().Get(name)
				if value == "" {
					continue
				}

				target, parseErr := req.URL.Parse(value)
				if parseErr != nil || (target.Host != "" && target.Host != req.Host) {
					continue
				}

				targets = append(targets, target)
			}

			// Invalidate related GET keys when the method is not allowed and the response is valid
			req.Method = http.MethodGet
			for _, target := range targets {
				invalidated := req.Clone(req.Context())
				invalidated.URL = target
				invalidated.RequestURI = target.RequestURI()

				keyname := s.context.SetContext(invalidated, rq).Context().Value(context.Key).(string)
				for _, storer := range s.Storers {
					storer.Delete(core.MappingKeyPrefix + keyname)
				}
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
	// Track whether the buffer ownership has been handed off to the background
	// goroutine. If so, we must not return it to the pool on exit because the
	// goroutine may still be writing to it.
	var bufPoolOwned atomic.Bool
	bufPoolOwned.Store(true)
	defer func() {
		if bufPoolOwned.Load() {
			bufPool.Reset()
			s.bufPool.Put(bufPool)
		}
	}()

	customWriter := NewCustomWriter(req, rw, bufPool)
	customWriter.Headers.Add("Range", req.Header.Get("Range"))
	req.Header.Del("Range")

	// Keep it while waiting for a confirmation that everything is fine.
	// if req.Context().Err() != nil {
	// 	// crw.mutex.Lock()
	// 	// crw.headersSent = true
	// 	// crw.mutex.Unlock()
	// }

	backfillIds := 0

	s.Configuration.GetLogger().Debugf("Request cache-control %+v", requestCc)
	// RFC 9111 section 5.2.1.5: a request carrying `no-store` is not answered
	// from the cache at all. `no-cache` on the other hand keeps the stored
	// response usable, provided it is revalidated first.
	if modeContext.Bypass_request || !requestCc.NoStore {
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

		now := req.Context().Value(context.Now).(time.Time)
		// A storer keeps a response around past the instant it turns stale so
		// it can still be revalidated, so its own fresh/stale split is looser
		// than the RFC one: the stored expiry is what decides whether the
		// response may be reused without asking the origin first.
		forceRevalidation := !modeContext.Bypass_request && requestCc.NoCache
		if fresh != nil && !forceRevalidation && !modeContext.Bypass_response {
			// A stored `no-cache` response may not be reused before the
			// origin confirmed it is still valid, so it goes down the same
			// road as a stale one and gets revalidated conditionally.
			storedControlName, _ := s.SurrogateKeyStorer.GetSurrogateControl(fresh.Header)
			if storedCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(fresh.Header, storedControlName)); storedCc != nil && storedCc.NoCachePresent {
				prometheus.Increment(prometheus.NoCachedResponseCounter)

				forceRevalidation = true
			}
		}

		if fresh != nil && (forceRevalidation || !rfc.IsStoredFresh(fresh.Header, now)) {
			fresh, stale = nil, fresh
		}

		if fresh != nil && (!modeContext.Strict || rfc.ValidateCacheControl(fresh, requestCc)) {
			freshClone := *fresh
			freshClone.Header = fresh.Header.Clone()

			go s.backfillStorers(backfillIds, cachedKey, req.Clone(req.Context()), &freshClone)

			response := fresh

			// This shortcut answers a conditional client request; an
			// unconditional one still has to go through the directives the
			// stored response carries.
			if validator.ResponseETag != "" && validator.Matched && len(validator.RequestETags) > 0 {
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
					_ = response.Body.Close()
				})
				_, _ = customWriter.Send()

				return nil
			}

			// RFC 9110 section 13.1.3: a fresh stored response that has not
			// been modified since the date the client presents can be
			// answered with a `304` without asking the origin.
			if validator.IfModifiedSincePresent && !validator.IfNoneMatchPresent {
				if lastModified, err := rfc.ParseHTTPDate(response.Header.Get("Last-Modified")); err == nil && !lastModified.After(validator.IfModifiedSince) {
					rfc.SetCacheStatusHeader(response, storerName)
					maps.Copy(customWriter.Header(), response.Header)
					customWriter.WriteHeader(http.StatusNotModified)
					customWriter.handleBuffer(func(b *bytes.Buffer) {
						b.Reset()
					})
					_, _ = customWriter.Send()

					return nil
				}
			}

			if !modeContext.Bypass_request && validator.NeedRevalidation {
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
					_ = response.Body.Close()
				})
				_, err := customWriter.Send()
				prometheus.Increment(prometheus.CachedResponseCounter)

				return err
			}
		} else if stale != nil {
			if handled, err := s.serveStale(stale, storerName, validator, next, customWriter, req, requestCc, cachedKey, uri, now); handled {
				return err
			}
		}
	}

	// RFC 9111 section 5.2.1.7: `only-if-cached` forbids forwarding the
	// request, so with nothing left to serve the cache answers itself.
	if !modeContext.Bypass_request && requestCc.OnlyIfCached {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=uri-miss; detail=ONLY-IF-CACHED")
		customWriter.WriteHeader(http.StatusGatewayTimeout)
		customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})
		_, err := customWriter.Send()

		return err
	}

	errorCacheCh := make(chan error, 1)

	go func(vr *http.Request, cw *CustomWriter) {
		prometheus.Increment(prometheus.NoCachedResponseCounter)
		err := s.Upstream(cw, vr, next, requestCc, cachedKey, uri, false)
		// If the parent already returned (context timeout), we own the buffer now.
		if !bufPoolOwned.Load() {
			bufPool.Reset()
			s.bufPool.Put(bufPool)
		}
		errorCacheCh <- err
	}(req, customWriter)

	select {
	case <-req.Context().Done():
		// Transfer buffer ownership to the goroutine so it can return the
		// buffer to the pool once Upstream finishes.
		bufPoolOwned.Store(false)
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

// headersKeptOnRevalidation are the stored header fields a `304` must not
// overwrite. The body that gets served is the stored one, so its framing
// stays authoritative, and the bookkeeping fields Souin adds itself are
// recomputed when the refreshed response is stored again.
var headersKeptOnRevalidation = map[string]bool{
	"Content-Length":       true,
	"Cache-Status":         true,
	rfc.StoredLengthHeader: true,
	rfc.StoredTTLHeader:    true,
	rfc.StoredExpiryHeader: true,
}

// refreshStoredHeaders applies the header fields of a `304` to the stored
// ones, as RFC 9111 section 4.3.4 requires.
func refreshStoredHeaders(stored, from http.Header) http.Header {
	refreshed := stored.Clone()
	// The stored response has just been validated, so whatever age it had
	// accumulated is gone unless the origin says otherwise.
	refreshed.Del("Age")

	for name, values := range from {
		if headersKeptOnRevalidation[http.CanonicalHeaderKey(name)] {
			continue
		}

		refreshed[http.CanonicalHeaderKey(name)] = values
	}

	removeHopByHopHeaders(refreshed)

	return refreshed
}

func writeCachedResponse(customWriter *CustomWriter, headers http.Header, statusCode int, body []byte) {
	current := customWriter.Header()
	for name := range current {
		delete(current, name)
	}
	maps.Copy(current, headers)

	customWriter.WriteHeader(statusCode)
	customWriter.handleBuffer(func(b *bytes.Buffer) {
		b.Reset()
		_, _ = b.Write(body)
	})
}

// serveStale decides what to do with a stored response that is no longer
// fresh: serve it as is when the request (`max-stale`) or the response
// (`stale-while-revalidate`, `stale-if-error`) allows it, and otherwise
// revalidate it with the origin before reusing it (RFC 9111 sections 4.2.4
// and 4.3). It reports whether it handled the request at all: a stale
// response with no validator and no reason to be served is left to the
// regular upstream path.
func (s *SouinBaseHandler) serveStale(
	stale *http.Response,
	storerName string,
	validator *core.Revalidator,
	next handlerFunc,
	customWriter *CustomWriter,
	rq *http.Request,
	requestCc *cacheobject.RequestCacheDirectives,
	cachedKey string,
	uri string,
	now time.Time,
) (bool, error) {
	modeContext := rq.Context().Value(context.Mode).(*context.ModeContext)
	headerName, _ := s.SurrogateKeyStorer.GetSurrogateControl(stale.Header)
	responseCc, _ := cacheobject.ParseResponseCacheControl(rfc.HeaderAllCommaSepValuesString(stale.Header, headerName))
	if responseCc == nil {
		responseCc = &cacheobject.ResponseCacheDirectives{}
	}

	staleness := rfc.StoredStaleness(stale.Header, now)
	// Past the configured stale window the stored response may no longer be
	// served, whatever the request asks for; it is only good to revalidate.
	withinStaleWindow := staleness <= s.Configuration.GetDefaultCache().GetStale()
	// `must-revalidate` and `no-cache` both forbid reusing the response
	// without contacting the origin first, whatever the client allows.
	mustRevalidate := responseCc.MustRevalidate || responseCc.NoCachePresent ||
		validator.NeedRevalidation || requestCc.NoCache

	storedBody := new(bytes.Buffer)
	_, _ = io.Copy(storedBody, stale.Body)
	_ = stale.Body.Close()
	storedHeaders := stale.Header.Clone()

	serveStored := func() (bool, error) {
		rfc.SetCacheStatusHeader(stale, storerName)
		rfc.HitStaleCache(&stale.Header)
		writeCachedResponse(customWriter, stale.Header, stale.StatusCode, storedBody.Bytes())
		_, err := customWriter.Send()

		return true, err
	}

	if !modeContext.Strict {
		if withinStaleWindow {
			return serveStored()
		}

		return false, nil
	}

	if requestCc.OnlyIfCached {
		if withinStaleWindow {
			return serveStored()
		}

		customWriter.WriteHeader(http.StatusGatewayTimeout)
		customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})
		_, err := customWriter.Send()

		return true, err
	}

	if !mustRevalidate {
		acceptsStale := requestCc.MaxStaleSet ||
			(requestCc.MaxStale > -1 && staleness <= time.Duration(requestCc.MaxStale)*time.Second)
		if withinStaleWindow && acceptsStale && rfc.ValidateCacheControl(stale, requestCc) {
			return serveStored()
		}

		// RFC 5861 section 3: within the stale-while-revalidate window the
		// stale response is served right away and refreshed in the
		// background. That window is the response's own, it does not depend
		// on how long this cache was configured to keep stale content.
		if staleness <= time.Duration(responseCc.StaleWhileRevalidate)*time.Second {
			handled, err := serveStored()
			backgroundWriter := NewCustomWriter(rq, customWriter.Rw, new(bytes.Buffer))
			go func(goRq *http.Request) {
				_ = s.Revalidate(validator, next, backgroundWriter, goRq, requestCc, cachedKey, uri)
			}(rq)

			return handled, err
		}
	}

	// RFC 9111 section 4.2.4: a stale response may stand in for an origin
	// that cannot be reached, unless the response forbids it. A
	// `stale-if-error` directive asks for it explicitly and overrides that
	// prohibition.
	// A shared cache reads `s-maxage` and `proxy-revalidate` as forbidding
	// stale content just like `must-revalidate` does (RFC 9111 sections
	// 5.2.2.9 and 5.2.2.10).
	forbidsStale := responseCc.MustRevalidate || responseCc.NoCachePresent ||
		responseCc.ProxyRevalidate || responseCc.SMaxAge >= 0
	staleIfError := responseCc.StaleIfError > -1 || requestCc.StaleIfError > 0
	canServeOnError := withinStaleWindow && (staleIfError || !forbidsStale)

	hasValidator := storedHeaders.Get("Etag") != "" || storedHeaders.Get("Last-Modified") != ""
	if !mustRevalidate && !hasValidator && !canServeOnError {
		// Nothing to revalidate with and no reason to keep the stored
		// response around for this request: a plain fetch it is. A response
		// under `must-revalidate` is not in that case, it has to go through a
		// revalidation that fails loudly.
		return false, nil
	}

	clientConditional := validator.IfNoneMatchPresent || validator.IfModifiedSincePresent

	// RFC 9111 section 4.3.1: revalidate with the validators of the stored
	// response so the origin can answer with a bare `304`.
	if etag := storedHeaders.Get("Etag"); etag != "" {
		// Entity-tags travel quoted, whatever shape the origin stored them
		// in, otherwise the validator it gets back is not the one it sent.
		if !strings.HasPrefix(etag, `"`) && !strings.HasPrefix(etag, "W/") {
			etag = `"` + etag + `"`
		}

		rq.Header.Set("If-None-Match", etag)
	}
	if lastModified := storedHeaders.Get("Last-Modified"); lastModified != "" {
		rq.Header.Set("If-Modified-Since", lastModified)
	}

	err := s.Revalidate(validator, next, customWriter, rq, requestCc, cachedKey, uri)
	statusCode := customWriter.GetStatusCode()

	if err != nil {
		if canServeOnError {
			rfc.SetCacheStatusHeader(stale, storerName)
			rfc.HitStaleCache(&stale.Header)
			stale.Header.Set("Cache-Status", stale.Header.Get("Cache-Status")+fmt.Sprintf("; fwd-status=%d", statusCode))
			writeCachedResponse(customWriter, stale.Header, stale.StatusCode, storedBody.Bytes())
			_, sendErr := customWriter.Send()

			return true, sendErr
		}

		customWriter.Rw.WriteHeader(http.StatusGatewayTimeout)
		customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})
		_, sendErr := customWriter.Send()

		return true, sendErr
	}

	if statusCode != http.StatusNotModified {
		_, sendErr := customWriter.Send()

		return true, sendErr
	}

	// RFC 9111 section 4.3.4: the stored response is refreshed with what the
	// `304` carries, then reused.
	refreshed := refreshStoredHeaders(storedHeaders, customWriter.Header())
	writeCachedResponse(customWriter, refreshed, stale.StatusCode, storedBody.Bytes())

	_ = s.Store(customWriter, rq, requestCc, cachedKey, uri)

	// A `304` may only reach the client when the client asked conditionally
	// itself; the one the origin just sent answers the validators this cache
	// added on its own behalf.
	if clientConditional && validator.Matched {
		customWriter.WriteHeader(http.StatusNotModified)
		customWriter.handleBuffer(func(b *bytes.Buffer) {
			b.Reset()
		})
	}

	_, sendErr := customWriter.Send()

	return true, sendErr
}
