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
			var prometheusConfiguration, souinConfiguration, securityConfiguration map[string]interface{}
			apiConfiguration := v.(map[string]interface{})
			for apiK, apiV := range apiConfiguration {
				switch apiK {
				case "prometheus":
					prometheusConfiguration = make(map[string]interface{})
					if apiV != nil && len(apiV.(string)) != 0 {
						prometheusConfiguration = apiV.(map[string]interface{})
					}
				case "souin":
					souinConfiguration = make(map[string]interface{})
					if apiV != nil && len(apiV.(string)) != 0 {
						souinConfiguration = apiV.(map[string]interface{})
					}
				case "security":
					securityConfiguration = make(map[string]interface{})
					if apiV != nil && len(apiV.(string)) != 0 {
						securityConfiguration = apiV.(map[string]interface{})
					}
				}
			}
			if prometheusConfiguration != nil {
				a.Prometheus = configurationtypes.APIEndpoint{}
				a.Prometheus.Enable = true
				if prometheusConfiguration["basepath"] != nil {
					a.Prometheus.BasePath = prometheusConfiguration["basepath"].(string)
				}
				if prometheusConfiguration["security"] != nil {
					a.Prometheus.Security = prometheusConfiguration["security"].(bool)
				}
			}
			if souinConfiguration != nil {
				a.Souin = configurationtypes.APIEndpoint{}
				a.Souin.Enable = true
				if souinConfiguration["basepath"] != nil {
					a.Souin.BasePath = souinConfiguration["basepath"].(string)
				}
				if souinConfiguration["security"] != nil {
					a.Souin.Security = souinConfiguration["security"].(bool)
				}
			}
			if securityConfiguration != nil {
				a.Security = configurationtypes.SecurityAPI{}
				a.Security.Enable = true
				if securityConfiguration["basepath"] != nil {
					a.Security.BasePath = securityConfiguration["basepath"].(string)
				}
				if securityConfiguration["users"] != nil {
					a.Security.Users = securityConfiguration["users"].([]configurationtypes.User)
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
				case "headers":
					dc.Headers = parseStringSlice(defaultCacheV)
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
			break
		case "log_level":
			configuration.LogLevel = v.(string)
			break
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
	req = s.Retriever.GetContext().Method.SetContext(req)
	if !(canHandle(req, s.Retriever)) {
		rw.Header().Set("Cache-Status", "Souin; fwd=uri-miss")
		s.next.ServeHTTP(rw, req)
		return
	}

	if b, h := s.HandleInternally(req); b {
		h(rw, req)
		return
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	req = s.Retriever.GetContext().SetContext(req)
	isMutation := req.Context().Value(souin_ctx.IsMutationRequest).(bool)
	if isMutation {
		rw.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
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
