package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/TykTechnologies/tyk/ctx"
	"github.com/cespare/xxhash/v2"
	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/pquerna/cachecontrol/cacheobject"
)

var definitions map[string]*souinInstance = make(map[string]*souinInstance)

func getInstanceFromRequest(r *http.Request) (s *souinInstance) {
	def := ctx.GetDefinition(r)
	var found bool
	if s, found = definitions[def.APIID]; !found {
		s = parseConfiguration(def.APIID, def.ConfigData)
	}

	return s
}

// SouinResponseHandler stores the response before sent to the client if possible, only returns otherwise
func SouinResponseHandler(rw http.ResponseWriter, rs *http.Response, baseRq *http.Request) {
	if rs.Header.Get("Cache-Status") != "" {
		return
	}
	baseRq.URL.Path = baseRq.RequestURI
	customWriter := NewCustomWriter(baseRq, rw, bytes.NewBuffer([]byte{}))
	s := getInstanceFromRequest(baseRq)
	rq := s.context.SetContext(s.context.SetBaseContext(baseRq), baseRq)
	cacheName := rq.Context().Value(context.CacheName).(string)
	if rq.Header.Get("Upgrade") == "websocket" || (s.SouinBaseHandler.ExcludeRegex != nil && s.SouinBaseHandler.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=EXCLUDED-REQUEST-URI")
		return
	}

	if !rq.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=UNSUPPORTED-METHOD")

		return
	}

	switch customWriter.statusCode {
	case 500, 502, 503, 504:
		return
	}

	responseCc, _ := cacheobject.ParseResponseCacheControl(customWriter.Header().Get("Cache-Control"))

	currentMatchedURL := s.SouinBaseHandler.DefaultMatchedUrl
	if regexpURL := s.SouinBaseHandler.RegexpUrls.FindString(rq.Host + rq.URL.Path); regexpURL != "" {
		u := s.SouinBaseHandler.Configuration.GetUrls()[regexpURL]
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
	date, _ := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	customWriter.Headers.Set(rfc.StoredTTLHeader, ma.String())
	ma = ma - time.Since(date)

	requestCc, coErr := cacheobject.ParseRequestCacheControl(rq.Header.Get("Cache-Control"))

	if coErr != nil || requestCc == nil {
		rs.Header.Set("Cache-Status", "Souin; fwd=bypass; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		return
	}

	status := fmt.Sprintf("%s; fwd=uri-miss", rq.Context().Value(context.CacheName))
	if !requestCc.NoStore && !responseCc.NoStore {
		_, _ = io.Copy(customWriter, rs.Body)
		rs.Body = io.NopCloser(bytes.NewBuffer(customWriter.Buf.Bytes()))
		res := http.Response{
			StatusCode: customWriter.statusCode,
			Body:       io.NopCloser(bytes.NewBuffer(customWriter.Buf.Bytes())),
			Header:     customWriter.Headers,
		}

		res.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		res.Request = rq
		response, err := httputil.DumpResponse(&res, true)
		cachedKey := rq.Context().Value(context.Key).(string)
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
					for _, storer := range s.SouinBaseHandler.Storers {
						wg.Add(1)
						go func(currentStorer types.Storer) {
							defer wg.Done()
							if currentStorer.Set(cachedKey, response, currentMatchedURL, ma) != nil {
								mu.Lock()
								fails = append(fails, fmt.Sprintf("; detail=%s-INSERTION-ERROR", currentStorer.Name()))
								mu.Unlock()
							}
						}(storer)
					}

					wg.Wait()
					if len(fails) < len(s.SouinBaseHandler.Storers) {
						go func(rs http.Response, key string) {
							_ = s.SouinBaseHandler.SurrogateKeyStorer.Store(&rs, key)
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
	rs.Header.Set("Cache-Status", status+"; key="+rfc.GetCacheKeyFromCtx(rq.Context()))
}

// SouinRequestHandler handle the Tyk request
func SouinRequestHandler(rw http.ResponseWriter, baseRq *http.Request) {
	s := getInstanceFromRequest(baseRq)

	if b, handler := s.SouinBaseHandler.HandleInternally(baseRq); b {
		handler(rw, baseRq)
		return
	}

	rq := s.context.SetBaseContext(baseRq)
	cacheName := rq.Context().Value(context.CacheName).(string)
	if rq.Header.Get("Upgrade") == "websocket" || (s.SouinBaseHandler.ExcludeRegex != nil && s.SouinBaseHandler.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=EXCLUDED-REQUEST-URI")
		return
	}

	if !rq.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=UNSUPPORTED-METHOD")

		return
	}

	requestCc, coErr := cacheobject.ParseRequestCacheControl(rq.Header.Get("Cache-Control"))

	modeContext := rq.Context().Value(context.Mode).(*context.ModeContext)
	if coErr != nil || requestCc == nil {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=bypass; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		return
	}

	rq = s.context.SetContext(rq, baseRq)
	cachedKey := rq.Context().Value(context.Key).(string)

	bufPool := s.bufPool.Get().(*bytes.Buffer)
	bufPool.Reset()
	defer s.bufPool.Put(bufPool)
	if !requestCc.NoCache {
		validator := rfc.ParseRequest(rq)
		var storerName string
		var fresh, stale *http.Response
		finalKey := cachedKey
		if rq.Context().Value(context.Hashed).(bool) {
			finalKey = fmt.Sprint(xxhash.Sum64String(finalKey))
		}
		for _, currentStorer := range s.Storers {
			fresh, stale = currentStorer.GetMultiLevel(finalKey, rq, validator)

			if fresh != nil || stale != nil {
				storerName = currentStorer.Name()
				fmt.Printf("Found at least one valid response in the %s storage\n", currentStorer.Name())
				break
			}
		}

		if fresh != nil && (!modeContext.Strict || rfc.ValidateCacheControl(fresh, requestCc)) {
			response := fresh
			rfc.SetCacheStatusHeader(response, storerName)
			if rfc.ValidateMaxAgeCachedResponse(requestCc, response) != nil {
				for hn, hv := range response.Header {
					rw.Header().Set(hn, strings.Join(hv, ", "))
				}
				_, _ = io.Copy(rw, response.Body)

				return
			}
		} else if !requestCc.OnlyIfCached && (requestCc.MaxStaleSet || requestCc.MaxStale > -1) {
			response := stale

			if nil != response && rfc.ValidateCacheControl(response, requestCc) {
				addTime, _ := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
				rfc.SetCacheStatusHeader(response, storerName)

				responseCc, _ := cacheobject.ParseResponseCacheControl(response.Header.Get("Cache-Control"))
				if responseCc.StaleIfError > 0 {
					h := response.Header
					rfc.HitStaleCache(&h)
					for hn, hv := range h {
						h.Set(hn, strings.Join(hv, ", "))
					}
					_, _ = io.Copy(rw, response.Body)

					return
				}

				if rfc.ValidateMaxAgeCachedStaleResponse(requestCc, response, int(addTime.Seconds())) != nil {
					h := response.Header
					rfc.HitStaleCache(&h)
					for hn, hv := range h {
						h.Set(hn, strings.Join(hv, ", "))
					}
					_, _ = io.Copy(rw, response.Body)

					return
				}
			}
		}
	}
}

type souinInstance struct {
	*middleware.SouinBaseHandler

	context *context.Context
	bufPool *sync.Pool
}

func init() {
	fmt.Println(`message="Souin configuration is now loaded."`)
}

func main() {}
