package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/TykTechnologies/tyk/ctx"
	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/pkg/rfc"
	"github.com/darkweak/souin/pkg/storage"
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
func SouinResponseHandler(rw http.ResponseWriter, rs *http.Response, rq *http.Request) {
	if rs.Header.Get("Cache-Status") != "" {
		return
	}
	customWriter := NewCustomWriter(rq, rw, bytes.NewBuffer([]byte{}))
	s := getInstanceFromRequest(rq)
	rq = s.context.SetContext(s.context.SetBaseContext(rq))
	cacheName := rq.Context().Value(context.CacheName).(string)
	if rq.Header.Get("Upgrade") == "websocket" || (s.ExcludeRegex != nil && s.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=uri-miss; detail=EXCLUDED-REQUEST-URI")
		return
	}

	if !rq.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=uri-miss; detail=UNSUPPORTED-METHOD")

		return
	}

	switch customWriter.statusCode {
	case 500, 502, 503, 504:
		return
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
	date, _ := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	customWriter.Headers.Set(rfc.StoredTTLHeader, ma.String())
	ma = ma - time.Since(date)

	requestCc, coErr := cacheobject.ParseRequestCacheControl(rq.Header.Get("Cache-Control"))

	if coErr != nil || requestCc == nil {
		rs.Header.Set("Cache-Status", "Souin; fwd=uri-miss; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		return
	}

	status := fmt.Sprintf("%s; fwd=uri-miss", rq.Context().Value(context.CacheName))
	if !requestCc.NoStore && !responseCc.NoStore {
		io.Copy(customWriter, rs.Body)
		rs.Body = ioutil.NopCloser(bytes.NewBuffer(customWriter.Buf.Bytes()))
		res := http.Response{
			StatusCode: customWriter.statusCode,
			Body:       ioutil.NopCloser(bytes.NewBuffer(customWriter.Buf.Bytes())),
			Header:     customWriter.Headers,
		}

		res.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
		res.Request = rq
		response, err := httputil.DumpResponse(&res, true)
		cachedKey := rq.Context().Value(context.Key).(string)
		if err == nil {
			variedHeaders := rfc.HeaderAllCommaSepValues(res.Header)
			cachedKey += rfc.GetVariedCacheKey(rq, variedHeaders)
			if s.Storer.Set(cachedKey, response, currentMatchedURL, ma) == nil {
				status += "; stored"
			} else {
				status += "; detail=STORAGE-INSERTION-ERROR"
			}
		}
	} else {
		status += "; detail=NO-STORE-DIRECTIVE"
	}
	rs.Header.Set("Cache-Status", status+"; key="+rfc.GetCacheKeyFromCtx(rq.Context()))
}

// SouinRequestHandler handle the Tyk request
func SouinRequestHandler(rw http.ResponseWriter, rq *http.Request) {
	s := getInstanceFromRequest(rq)

	if b, handler := s.HandleInternally(rq); b {
		handler(rw, rq)
		return
	}

	rq = s.context.SetBaseContext(rq)
	cacheName := rq.Context().Value(context.CacheName).(string)
	if rq.Header.Get("Upgrade") == "websocket" || (s.ExcludeRegex != nil && s.ExcludeRegex.MatchString(rq.RequestURI)) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=uri-miss; detail=EXCLUDED-REQUEST-URI")
		return
	}

	if !rq.Context().Value(context.SupportedMethod).(bool) {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=uri-miss; detail=UNSUPPORTED-METHOD")

		return
	}

	requestCc, coErr := cacheobject.ParseRequestCacheControl(rq.Header.Get("Cache-Control"))

	if coErr != nil || requestCc == nil {
		rw.Header().Set("Cache-Status", cacheName+"; fwd=uri-miss; detail=CACHE-CONTROL-EXTRACTION-ERROR")

		return
	}

	rq = s.context.SetContext(rq)
	cachedKey := rq.Context().Value(context.Key).(string)

	bufPool := s.bufPool.Get().(*bytes.Buffer)
	bufPool.Reset()
	defer s.bufPool.Put(bufPool)
	if !requestCc.NoCache {
		cachedVal := s.Storer.Prefix(cachedKey, rq)
		response, _ := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(cachedVal)), rq)

		if response != nil && rfc.ValidateCacheControl(response) {
			rfc.SetCacheStatusHeader(response)
			if rfc.ValidateMaxAgeCachedResponse(requestCc, response) != nil {
				for hn, hv := range response.Header {
					rw.Header().Set(hn, strings.Join(hv, ", "))
				}
				io.Copy(rw, response.Body)

				return
			}
		} else if response == nil {
			staleCachedVal := s.Storer.Prefix(storage.StalePrefix+cachedKey, rq)
			response, _ = http.ReadResponse(bufio.NewReader(bytes.NewBuffer(staleCachedVal)), rq)
			if nil != response && rfc.ValidateCacheControl(response) {
				addTime, _ := time.ParseDuration(response.Header.Get(rfc.StoredTTLHeader))
				rfc.SetCacheStatusHeader(response)

				responseCc, _ := cacheobject.ParseResponseCacheControl(response.Header.Get("Cache-Control"))
				if responseCc.StaleIfError > 0 {
					h := response.Header
					rfc.HitStaleCache(&h)
					for hn, hv := range h {
						h.Set(hn, strings.Join(hv, ", "))
					}
					io.Copy(rw, response.Body)

					return
				}

				if rfc.ValidateMaxAgeCachedStaleResponse(requestCc, response, int(addTime.Seconds())) != nil {
					h := response.Header
					rfc.HitStaleCache(&h)
					for hn, hv := range h {
						h.Set(hn, strings.Join(hv, ", "))
					}
					io.Copy(rw, response.Body)

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
