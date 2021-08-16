package souin

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/rfc"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SouinTraefikPlugin declaration.
type SouinTraefikPlugin struct {
	next http.Handler
	name string
	SouinBasePlugin
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type TestConfiguration map[string]interface{}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *TestConfiguration {
	return &TestConfiguration{}
}

func parseConfiguration(c map[string]interface{}) Configuration {
	configuration := Configuration{}

	for k, v := range c {
		switch k {
		case "default_cache":
			dc := configurationtypes.DefaultCache{
				Distributed: false,
				Headers:     []string{},
				Olric: configurationtypes.CacheProvider{
					URL:           "",
					Path:          "",
					Configuration: nil,
				},
				Regex: configurationtypes.Regex{},
				TTL:   configurationtypes.Duration{},
			}
			defaultCache := v.(map[string]interface{})
			for defaultCacheK, defaultCacheV := range defaultCache {
				switch defaultCacheK {
				case "headers":
					dc.Headers = strings.Split(defaultCacheV.(string), ",")
				case "regex":
					exclude := defaultCacheV.(map[string]interface{})["exclude"].(string)
					if exclude != "" {
						dc.Regex = configurationtypes.Regex{Exclude: exclude}
					}
				case "ttl":
					ttl, err := time.ParseDuration(defaultCacheV.(string))
					if err == nil {
						dc.TTL = configurationtypes.Duration{Duration: ttl}
					}
				}
			}
			configuration.DefaultCache = &dc
			break
		case "log_level":
			configuration.LogLevel = v.(string)
			break
		case "urls":
			u := make(map[string]configurationtypes.URL)
			urls := v.(map[string]interface{})

			for urlK, urlV := range urls {
				currentUrl := configurationtypes.URL{
					TTL:     configurationtypes.Duration{},
					Headers: nil,
				}
				currentValue := urlV.(map[string]interface{})
				currentUrl.Headers = strings.Split(currentValue["headers"].(string), ",")
				d := currentValue["ttl"].(string)
				ttl, err := time.ParseDuration(d)
				if err == nil {
					currentUrl.TTL = configurationtypes.Duration{Duration: ttl}
				}
				u[urlK] = currentUrl
			}
			configuration.URLs = u
		case "ykeys":
			ykeys := make(map[string]configurationtypes.YKey)
			d, _ := json.Marshal(v)
			_ = json.Unmarshal(d, &ykeys)
			configuration.Ykeys = ykeys
		}
	}

	return configuration
}

// New create Souin instance.
func New(_ context.Context, next http.Handler, config *TestConfiguration, name string) (http.Handler, error) {
	s := &SouinTraefikPlugin{
		name: name,
		next: next,
	}
	c := parseConfiguration(*config)

	s.Retriever = DefaultSouinPluginInitializerFromConfiguration(&c)
	return s, nil
}

type getterContext struct {
	rw   http.ResponseWriter
	req  *http.Request
	next http.Handler
}

type customWriter struct {
	response *http.Response
	http.ResponseWriter
}

func (r *customWriter) Write(b []byte) (int, error) {
	r.response.Header = r.ResponseWriter.Header()
	r.response.StatusCode = http.StatusOK
	r.response.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return r.ResponseWriter.Write(b)
}

type key string

const getterContextCtxKey key = "getter_context"

func (s *SouinTraefikPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	customRW := &customWriter{
		ResponseWriter: rw,
		response: &http.Response{},
	}
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	getterCtx := getterContext{rw, req, s.next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	path := req.Host + req.URL.Path
	regexpURL := s.Retriever.GetRegexpUrls().FindString(path)
	url := configurationtypes.URL{
		TTL:     configurationtypes.Duration{Duration: s.Retriever.GetConfiguration().GetDefaultCache().GetTTL()},
		Headers: s.Retriever.GetConfiguration().GetDefaultCache().GetHeaders(),
	}
	if "" != regexpURL {
		u := s.Retriever.GetConfiguration().GetUrls()[regexpURL]
		if u.TTL.Duration != 0 {
			url.TTL = u.TTL
		}
		if len(u.Headers) != 0 {
			url.Headers = u.Headers
		}
	}
	s.Retriever.GetTransport().SetURL(url)
	s.Retriever.SetMatchedURL(url)

	headers := ""
	if s.Retriever.GetMatchedURL().Headers != nil && len(s.Retriever.GetMatchedURL().Headers) > 0 {
		for _, h := range s.Retriever.GetMatchedURL().Headers {
			headers += strings.ReplaceAll(req.Header.Get(h), " ", "")
		}
	}

	DefaultSouinPluginCallback(rw, req, s.Retriever, s.RequestCoalescing, func(_ http.ResponseWriter, _ *http.Request) error {
		s.next.ServeHTTP(customRW, req)
		req.Response = customRW.response

		_, e := s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(req)

		return e
	})
}
