package traefik

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
)

// SouinTraefikMiddleware declaration.
type SouinTraefikMiddleware struct {
	next http.Handler
	name string
	*middleware.SouinBaseHandler
}

// TestConfiguration is the temporary configuration for Træfik
type TestConfiguration map[string]interface{}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *TestConfiguration {
	return &TestConfiguration{}
}

func configCacheKey(keyConfiguration map[string]interface{}) configurationtypes.Key {
	key := configurationtypes.Key{}

	for keyK, keyV := range keyConfiguration {
		switch keyK {
		case "disable_body":
			key.DisableBody = parseBool(keyV)
		case "disable_host":
			key.DisableHost = parseBool(keyV)
		case "disable_method":
			key.DisableMethod = parseBool(keyV)
		case "disable_query":
			key.DisableQuery = parseBool(keyV)
		case "headers":
			key.Headers = parseStringSlice(keyV)
		case "hide":
			key.Hide = parseBool(keyV)
		}
	}

	return key
}

func parseBool(v interface{}) bool {
	if v != nil {
		boolValue, err := strconv.ParseBool(v.(string))
		if err != nil && boolValue {
			return true
		}
	}

	return false
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
		case "cache_keys":
			cacheKeys := make(configurationtypes.CacheKeys, 0)
			cacheKeyConfiguration := v.(map[string]interface{})
			for cacheKeyConfigurationK, cacheKeyConfigurationV := range cacheKeyConfiguration {
				cacheKeyK := configurationtypes.RegValue{
					Regexp: regexp.MustCompile(cacheKeyConfigurationK),
				}
				cacheKeyV := configCacheKey(cacheKeyConfigurationV.(map[string]interface{}))
				cacheKeys = append(cacheKeys, configurationtypes.CacheKey{
					cacheKeyK: cacheKeyV,
				})
			}
			configuration.CacheKeys = cacheKeys
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
				case "key":
					dc.Key = configCacheKey(defaultCacheV.(map[string]interface{}))
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
	if value, ok := i.([]string); ok {
		return value
	}
	if value, ok := i.([]interface{}); ok {
		var arr []string
		for _, v := range value {
			arr = append(arr, v.(string))
		}
		return arr
	}

	if value, ok := i.(string); ok {
		if strings.HasPrefix(value, "║24║") {
			return strings.Split(strings.TrimPrefix(value, "║24║"), "║")
		}
		return strings.Split(value, ",")
	}

	if value, ok := i.([]string); ok {
		return value
	}

	return nil
}

// New create Souin instance.
func New(_ context.Context, next http.Handler, config *TestConfiguration, name string) (http.Handler, error) {
	c := parseConfiguration(*config)

	return &SouinTraefikMiddleware{
		name:             name,
		next:             next,
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}, nil
}

func (s *SouinTraefikMiddleware) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Printf("%+v\n", s.SouinBaseHandler)
	_ = s.SouinBaseHandler.ServeHTTP(rw, req, func(w http.ResponseWriter, r *http.Request) error {
		s.next.ServeHTTP(w, r)

		return nil
	})
}
