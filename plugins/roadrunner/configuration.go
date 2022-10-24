package roadrunner

import (
	"regexp"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/plugins"
)

const (
	configurationKey = "http.cache"
	path             = "path"
	url              = "url"
	configurationPK  = "configuration"
)

func parseAPI(apiConfiguration map[string]interface{}) configurationtypes.API {
	var a configurationtypes.API
	var prometheusConfiguration, souinConfiguration map[string]interface{}

	for apiK, apiV := range apiConfiguration {
		switch apiK {
		case "basepath":
			a.BasePath, _ = apiV.(string)
		case "prometheus":
			prometheusConfiguration, _ = apiV.(map[string]interface{})
		case "souin":
			souinConfiguration, _ = apiV.(map[string]interface{})
		}
	}
	if prometheusConfiguration != nil {
		a.Prometheus = configurationtypes.APIEndpoint{}
		a.Prometheus.Enable = true
		if prometheusConfiguration["basepath"] != nil {
			a.Prometheus.BasePath, _ = prometheusConfiguration["basepath"].(string)
		}
	}
	if souinConfiguration != nil {
		a.Souin = configurationtypes.APIEndpoint{}
		a.Souin.Enable = true
		if souinConfiguration["basepath"] != nil {
			a.Souin.BasePath, _ = souinConfiguration["basepath"].(string)
		}
	}

	return a
}

func parseCacheKeys(ccConfiguration map[string]interface{}) map[configurationtypes.RegValue]configurationtypes.Key {
	cacheKeys := make(map[configurationtypes.RegValue]configurationtypes.Key)
	for cacheKeysConfigurationK, cacheKeysConfigurationV := range ccConfiguration {
		ck := configurationtypes.Key{}
		for cacheKeysConfigurationVMapK := range cacheKeysConfigurationV.(map[string]interface{}) {
			switch cacheKeysConfigurationVMapK {
			case "disable_body":
				ck.DisableBody = true
			case "disable_host":
				ck.DisableHost = true
			case "disable_method":
				ck.DisableMethod = true
			}
		}
		rg := regexp.MustCompile(cacheKeysConfigurationK)
		cacheKeys[configurationtypes.RegValue{Regexp: rg}] = ck
	}

	return cacheKeys
}

func parseDefaultCache(dcConfiguration map[string]interface{}) *configurationtypes.DefaultCache {
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
	for defaultCacheK, defaultCacheV := range dcConfiguration {
		switch defaultCacheK {
		case "allowed_http_verbs":
			dc.AllowedHTTPVerbs = make([]string, 0)
			if values, ok := defaultCacheV.([]string); ok {
				dc.AllowedHTTPVerbs = values
			} else if values, ok := defaultCacheV.([]interface{}); ok {
				for _, v := range values {
					dc.AllowedHTTPVerbs = append(dc.AllowedHTTPVerbs, v.(string))
				}
			}
		case "badger":
			provider := configurationtypes.CacheProvider{}
			for badgerConfigurationK, badgerConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch badgerConfigurationK {
				case url:
					provider.URL, _ = badgerConfigurationV.(string)
				case path:
					provider.Path, _ = badgerConfigurationV.(string)
				case configurationPK:
					provider.Configuration = badgerConfigurationV.(map[string]interface{})
				}
			}
			dc.Badger = provider
		case "cache_name":
			dc.CacheName, _ = defaultCacheV.(string)
		case "cdn":
			cdn := configurationtypes.CDN{}
			for cdnConfigurationK, cdnConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch cdnConfigurationK {
				case "api_key":
					cdn.APIKey, _ = cdnConfigurationV.(string)
				case "dynamic":
					cdn.Dynamic = true
				case "hostname":
					cdn.Hostname, _ = cdnConfigurationV.(string)
				case "network":
					cdn.Network, _ = cdnConfigurationV.(string)
				case "provider":
					cdn.Provider, _ = cdnConfigurationV.(string)
				case "strategy":
					cdn.Strategy, _ = cdnConfigurationV.(string)
				}
			}
			dc.CDN = cdn
		case "etcd":
			provider := configurationtypes.CacheProvider{}
			dc.Distributed = true
			for etcdConfigurationK, etcdConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch etcdConfigurationK {
				case url:
					provider.URL, _ = etcdConfigurationV.(string)
				case path:
					provider.Path, _ = etcdConfigurationV.(string)
				case configurationPK:
					provider.Configuration = etcdConfigurationV.(map[string]interface{})
				}
			}
			dc.Etcd = provider
		case "headers":
			dc.Headers = make([]string, 0)
			if values, ok := defaultCacheV.([]string); ok {
				dc.Headers = values
			} else if values, ok := defaultCacheV.([]interface{}); ok {
				for _, v := range values {
					dc.Headers = append(dc.Headers, v.(string))
				}
			}
		case "nuts":
			provider := configurationtypes.CacheProvider{}
			for nutsConfigurationK, nutsConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch nutsConfigurationK {
				case url:
					provider.URL, _ = nutsConfigurationV.(string)
				case path:
					provider.Path, _ = nutsConfigurationV.(string)
				case configurationPK:
					provider.Configuration = nutsConfigurationV.(map[string]interface{})
				}
			}
			dc.Nuts = provider
		case "olric":
			provider := configurationtypes.CacheProvider{}
			dc.Distributed = true
			for olricConfigurationK, olricConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch olricConfigurationK {
				case url:
					provider.URL, _ = olricConfigurationV.(string)
				case path:
					provider.Path, _ = olricConfigurationV.(string)
				case configurationPK:
					provider.Configuration = olricConfigurationV.(map[string]interface{})
				}
			}
			dc.Distributed = true
			dc.Olric = provider
		case "redis":
			provider := configurationtypes.CacheProvider{}
			dc.Distributed = true
			for redisConfigurationK, redisConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch redisConfigurationK {
				case url:
					provider.URL, _ = redisConfigurationV.(string)
				case path:
					provider.Path, _ = redisConfigurationV.(string)
				case configurationPK:
					provider.Configuration = redisConfigurationV.(map[string]interface{})
				}
			}
			dc.Distributed = true
			dc.Redis = provider
		case "regex":
			v := defaultCacheV.(map[string]interface{})
			if v["exclude"] != "" {
				dc.Regex = configurationtypes.Regex{Exclude: v["exclude"].(string)}
			}
		case "timeout":
			timeout := configurationtypes.Timeout{}
			for timeoutK, timeoutV := range defaultCacheV.(map[string]interface{}) {
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
		case "stale":
			ttl, err := time.ParseDuration(defaultCacheV.(string))
			if err == nil {
				dc.Stale = configurationtypes.Duration{Duration: ttl}
			}
		case "default_cache_control":
			dc.DefaultCacheControl, _ = defaultCacheV.(string)
		}
	}

	return &dc
}

func parseURLs(urls map[string]interface{}) map[string]configurationtypes.URL {
	u := make(map[string]configurationtypes.URL)

	for urlK, urlV := range urls {
		currentURL := configurationtypes.URL{
			TTL:     configurationtypes.Duration{},
			Headers: nil,
		}

		for k, v := range urlV.(map[string]interface{}) {
			switch k {
			case "headers":
				currentURL.Headers = make([]string, 0)
				if values, ok := urlV.([]string); ok {
					currentURL.Headers = values
				} else if values, ok := urlV.([]interface{}); ok {
					for _, value := range values {
						currentURL.Headers = append(currentURL.Headers, value.(string))
					}
				}
			case "ttl":
				if ttl, err := time.ParseDuration(v.(string)); err == nil {
					currentURL.TTL = configurationtypes.Duration{Duration: ttl}
				}
			case "default_cache_control":
				currentURL.DefaultCacheControl, _ = v.(string)
			}
		}
		u[urlK] = currentURL
	}

	return u
}

func parseSurrogateKeys(surrogates map[string]interface{}) map[string]configurationtypes.SurrogateKeys {
	u := make(map[string]configurationtypes.SurrogateKeys)

	for surrogateK, surrogateV := range surrogates {
		surrogate := configurationtypes.SurrogateKeys{}
		for key, value := range surrogateV.(map[string]interface{}) {
			switch key {
			case "headers":
				surrogate.Headers = make(map[string]string)
				for k, v := range value.(map[string]interface{}) {
					surrogate.Headers[k] = v.(string)
				}
			case "url":
				surrogate.URL = value.(string)
			}
		}
		u[surrogateK] = surrogate
	}

	return u
}

// ParseConfiguration parse the Roadrunner configuration into a valid HTTP
// cache configuration object.
func parseConfiguration(cfg Configurer) plugins.BaseConfiguration {
	var configuration plugins.BaseConfiguration

	for key, v := range cfg.Get(configurationKey).(map[string]interface{}) {
		switch key {
		case "api":
			configuration.API = parseAPI(v.(map[string]interface{}))
		case "cache_keys":
			configuration.CacheKeys = parseCacheKeys(v.(map[string]interface{}))
		case "default_cache":
			configuration.DefaultCache = parseDefaultCache(v.(map[string]interface{}))
		case "log_level":
			configuration.LogLevel, _ = v.(string)
		case "urls":
			configuration.URLs = parseURLs(v.(map[string]interface{}))
		case "ykeys":
			configuration.Ykeys = parseSurrogateKeys(v.(map[string]interface{}))
		case "surrogate_keys":
			configuration.SurrogateKeys = parseSurrogateKeys(v.(map[string]interface{}))
		}
	}

	return configuration
}
