package traefik

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/configurationtypes"
	souin_ctx "github.com/darkweak/souin/context"
	"github.com/darkweak/souin/rfc"
)

// SouinTraefikPlugin declaration.
type SouinTraefikPlugin struct {
	next http.Handler
	name string
	SouinBasePlugin
}

// TestConfiguration is the temporary configuration for Træfik
type TestConfiguration map[string]interface{}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *TestConfiguration {
	return &TestConfiguration{}
}

func parseConfiguration(c map[string]interface{}) Configuration {
	configuration := Configuration{}

	for k, v := range c {
		switch k {
		case "api":
			var a configurationtypes.API
			var prometheusConfiguration, souinConfiguration map[string]interface{}
			apiConfiguration := v.(map[string]interface{})
			for apiK, apiV := range apiConfiguration {
				switch apiK {
				case "prometheus":
					prometheusConfiguration = make(map[string]interface{})
					if apiV != nil {
						prometheus, ok := apiV.(map[string]interface{})
						if ok && len(prometheus) != 0 {
							prometheusConfiguration = apiV.(map[string]interface{})
						}
					}
				case "souin":
					souinConfiguration = make(map[string]interface{})
					if apiV != nil {
						souin, ok := apiV.(map[string]interface{})
						if ok && len(souin) != 0 {
							souinConfiguration = apiV.(map[string]interface{})
						}
					}
				}
			}
			if prometheusConfiguration != nil {
				a.Prometheus = configurationtypes.APIEndpoint{}
				a.Prometheus.Enable = true
				if prometheusConfiguration["basepath"] != nil {
					a.Prometheus.BasePath = prometheusConfiguration["basepath"].(string)
				}
			}
			if souinConfiguration != nil {
				a.Souin = configurationtypes.APIEndpoint{}
				a.Souin.Enable = true
				if souinConfiguration["basepath"] != nil {
					a.Souin.BasePath = souinConfiguration["basepath"].(string)
				}
			}
			configuration.API = a
		case "default_cache":
			dc := configurationtypes.DefaultCache{
				Distributed: false,
				Headers:     []string{},
				Olric: configurationtypes.CacheProvider{
					URL:           "",
					Path:          "",
					Configuration: nil,
				},
				Regex:               configurationtypes.Regex{},
				TTL:                 configurationtypes.Duration{},
				DefaultCacheControl: "",
			}
			defaultCache := v.(map[string]interface{})
			for defaultCacheK, defaultCacheV := range defaultCache {
				switch defaultCacheK {
				case "cache_name":
					dc.CacheName = defaultCacheV.(string)
				case "cdn":
					cdn := configurationtypes.CDN{
						Dynamic: true,
					}
					cdnConfiguration := defaultCacheV.(map[string]interface{})
					for cdnK, cdnV := range cdnConfiguration {
						switch cdnK {
						case "api_key":
							cdn.APIKey = cdnV.(string)
						case "dynamic":
							cdn.Dynamic = cdnV.(bool)
						case "email":
							cdn.Email = cdnV.(string)
						case "hostname":
							cdn.Hostname = cdnV.(string)
						case "network":
							cdn.Network = cdnV.(string)
						case "provider":
							cdn.Provider = cdnV.(string)
						case "service_id":
							cdn.ServiceID = cdnV.(string)
						case "strategy":
							cdn.Strategy = cdnV.(string)
						case "zone_id":
							cdn.ZoneID = cdnV.(string)
						}
					}
					dc.CDN = cdn
				case "headers":
					dc.Headers = parseStringSlice(defaultCacheV)
				case "regex":
					exclude := defaultCacheV.(map[string]interface{})["exclude"].(string)
					if exclude != "" {
						dc.Regex = configurationtypes.Regex{Exclude: exclude}
					}
				case "timeout":
					timeout := configurationtypes.Timeout{}
					timeoutConfiguration := defaultCacheV.(map[string]interface{})
					for timeoutK, timeoutV := range timeoutConfiguration {
						switch timeoutK {
						case "backend":
							d := configurationtypes.Duration{}
							ttl, err := time.ParseDuration(timeoutV.(string))
							if err == nil {
								d.Duration = ttl
							}
							timeout.Backend = d
						case "cache":
							d := configurationtypes.Duration{}
							ttl, err := time.ParseDuration(timeoutV.(string))
							if err == nil {
								d.Duration = ttl
							}
							timeout.Cache = d
						}
					}
					dc.Timeout = timeout
				case "ttl":
					ttl, err := time.ParseDuration(defaultCacheV.(string))
					if err == nil {
						dc.TTL = configurationtypes.Duration{Duration: ttl}
					}
				case "allowed_http_verbs":
					dc.AllowedHTTPVerbs = parseStringSlice(defaultCacheV)
				case "stale":
					stale, err := time.ParseDuration(defaultCacheV.(string))
					if err == nil {
						dc.Stale = configurationtypes.Duration{Duration: stale}
					}
				case "default_cache_control":
					dc.DefaultCacheControl = defaultCacheV.(string)
				}
			}
			configuration.DefaultCache = &dc
		case "log_level":
			configuration.LogLevel = v.(string)
		case "urls":
			u := make(map[string]configurationtypes.URL)
			urls := v.(map[string]interface{})

			for urlK, urlV := range urls {
				currentURL := configurationtypes.URL{
					TTL:     configurationtypes.Duration{},
					Headers: nil,
				}
				currentValue := urlV.(map[string]interface{})
				currentURL.Headers = parseStringSlice(currentValue["headers"])
				d := currentValue["ttl"].(string)
				ttl, err := time.ParseDuration(d)
				if err == nil {
					currentURL.TTL = configurationtypes.Duration{Duration: ttl}
				}
				if _, exists := currentValue["default_cache_control"]; exists {
					currentURL.DefaultCacheControl = currentValue["default_cache_control"].(string)
				}
				u[urlK] = currentURL
			}
			configuration.URLs = u
		case "ykeys":
			ykeys := make(map[string]configurationtypes.SurrogateKeys)
			d, _ := json.Marshal(v)
			_ = json.Unmarshal(d, &ykeys)
			configuration.Ykeys = ykeys
		}
	}

	return configuration
}

// parseStringSlice returns the string slice corresponding to the given interface.
// The interface can be of type string which contains a comma separated list of values (e.g. foo,bar) or of type []string.
func parseStringSlice(i interface{}) []string {
	if value, ok := i.(string); ok {
		return strings.Split(value, ",")
	}

	if value, ok := i.([]string); ok {
		return value
	}

	return nil
}

// New create Souin instance.
func New(_ context.Context, next http.Handler, config *TestConfiguration, name string) (http.Handler, error) {
	s := &SouinTraefikPlugin{
		name: name,
		next: next,
	}
	c := parseConfiguration(*config)

	s.Retriever = DefaultSouinPluginInitializerFromConfiguration(&c)
	s.MapHandler = api.GenerateHandlerMap(&c, s.Retriever.GetTransport())
	return s, nil
}

func (s *SouinTraefikPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	req = s.Retriever.GetContext().SetBaseContext(req)
	if !(canHandle(req, s.Retriever)) {
		rfc.MissCache(rw.Header().Set, req, "CANNOT-HANDLE")
		s.next.ServeHTTP(rw, req)
		return
	}

	if s.MapHandler != nil && s.MapHandler.Handlers != nil {
		for k, souinHandler := range *s.MapHandler.Handlers {
			if strings.Contains(req.RequestURI, k) {
				souinHandler(rw, req)
				return
			}
		}
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	req = s.Retriever.GetContext().SetContext(req)
	isMutation := req.Context().Value(souin_ctx.IsMutationRequest).(bool)
	if isMutation {
		rfc.MissCache(rw.Header().Set, req, "IS-MUTATION-REQUEST")
		s.next.ServeHTTP(rw, req)
		return
	}

	getterCtx := getterContext{rw, req, s.next}
	ctx := context.WithValue(req.Context(), getterContextCtxKey, getterCtx)
	req = req.WithContext(ctx)
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	customRW := &CustomWriter{
		Buf:      buf,
		Rw:       rw,
		Response: &http.Response{},
	}

	customRW.Buf = buf
	regexpURL := s.Retriever.GetRegexpUrls().FindString(req.Host + req.URL.Path)
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

	DefaultSouinPluginCallback(customRW, req, s.Retriever, s.RequestCoalescing, func(_ http.ResponseWriter, _ *http.Request) error {
		var e error
		s.next.ServeHTTP(customRW, req)
		req.Response = customRW.Response

		if req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(req); e != nil {
			return e
		}

		_, e = customRW.Send()
		return e
	})
}
