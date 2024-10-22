package agnostic

import (
	"regexp"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/middleware"
)

const (
	path            string = "path"
	url             string = "url"
	configurationPK string = "configuration"
)

func parseAPI(apiConfiguration map[string]interface{}) configurationtypes.API {
	var a configurationtypes.API
	var debugConfiguration, prometheusConfiguration, souinConfiguration map[string]interface{}

	for apiK, apiV := range apiConfiguration {
		switch apiK {
		case "basepath":
			a.BasePath = apiV.(string)
		case "debug":
			debugConfiguration, _ = apiV.(map[string]interface{})
		case "prometheus":
			prometheusConfiguration, _ = apiV.(map[string]interface{})
		case "souin":
			souinConfiguration, _ = apiV.(map[string]interface{})
		}
	}
	if debugConfiguration != nil {
		a.Debug = configurationtypes.APIEndpoint{}
		a.Debug.Enable = true
		if debugConfiguration["basepath"] != nil {
			a.Debug.BasePath, _ = debugConfiguration["basepath"].(string)
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

func parseCacheKeys(ccConfiguration map[string]interface{}) configurationtypes.CacheKeys {
	cacheKeys := make(configurationtypes.CacheKeys, 0)
	for cacheKeysConfigurationK, cacheKeysConfigurationV := range ccConfiguration {
		ck := configurationtypes.Key{}
		for cacheKeysConfigurationVMapK, cacheKeysConfigurationVMapV := range cacheKeysConfigurationV.(map[string]interface{}) {
			switch cacheKeysConfigurationVMapK {
			case "disable_body":
				ck.DisableBody = true
			case "disable_host":
				ck.DisableHost = true
			case "disable_method":
				ck.DisableMethod = true
			case "disable_query":
				ck.DisableQuery = true
			case "disable_scheme":
				ck.DisableScheme = true
			case "hash":
				ck.Hash = true
			case "hide":
				ck.Hide = true
			case "headers":
				if headers, ok := cacheKeysConfigurationVMapV.([]string); ok {
					ck.Headers = headers
				} else {
					for _, hv := range cacheKeysConfigurationVMapV.([]interface{}) {
						ck.Headers = append(ck.Headers, hv.(string))
					}
				}
			case "template":
				ck.Template = cacheKeysConfigurationVMapV.(string)
			}
		}
		cacheKeys = append(cacheKeys, configurationtypes.CacheKey{{Regexp: regexp.MustCompile(cacheKeysConfigurationK)}: ck})
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
			if verbs, ok := defaultCacheV.([]string); ok {
				dc.AllowedHTTPVerbs = verbs
			} else {
				for _, verb := range defaultCacheV.([]interface{}) {
					dc.AllowedHTTPVerbs = append(dc.AllowedHTTPVerbs, verb.(string))
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
			cdn := configurationtypes.CDN{
				Dynamic: true,
			}
			for cdnConfigurationK, cdnConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch cdnConfigurationK {
				case "api_key":
					cdn.APIKey, _ = cdnConfigurationV.(string)
				case "dynamic":
					cdn.Dynamic = cdnConfigurationV.(bool)
				case "email":
					cdn.Email, _ = cdnConfigurationV.(string)
				case "hostname":
					cdn.Hostname, _ = cdnConfigurationV.(string)
				case "network":
					cdn.Network, _ = cdnConfigurationV.(string)
				case "provider":
					cdn.Provider, _ = cdnConfigurationV.(string)
				case "service_id":
					cdn.ServiceID, _ = cdnConfigurationV.(string)
				case "strategy":
					cdn.Strategy, _ = cdnConfigurationV.(string)
				case "zone_id":
					cdn.ZoneID, _ = cdnConfigurationV.(string)
				}
			}
			dc.CDN = cdn
		case "etcd":
			provider := configurationtypes.CacheProvider{}
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
			if headers, ok := defaultCacheV.([]string); ok {
				dc.Headers = headers
			} else {
				if headers, ok := defaultCacheV.([]string); ok {
					dc.Headers = headers
				} else {
					for _, hv := range defaultCacheV.([]interface{}) {
						dc.Headers = append(dc.Headers, hv.(string))
					}
				}
			}
		case "mode":
			dc.Mode, _ = defaultCacheV.(string)
		case "nats":
			provider := configurationtypes.CacheProvider{}
			for natsConfigurationK, natsConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch natsConfigurationK {
				case url:
					provider.URL, _ = natsConfigurationV.(string)
				case configurationPK:
					provider.Configuration = natsConfigurationV.(map[string]interface{})
				}
			}
			dc.Nats = provider
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
		case "otter":
			provider := configurationtypes.CacheProvider{}
			for otterConfigurationK, otterConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch otterConfigurationK {
				case url:
					provider.URL, _ = otterConfigurationV.(string)
				case path:
					provider.Path, _ = otterConfigurationV.(string)
				case configurationPK:
					provider.Configuration = otterConfigurationV.(map[string]interface{})
				}
			}
			dc.Otter = provider
		case "olric":
			provider := configurationtypes.CacheProvider{}
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
			dc.Regex = configurationtypes.Regex{}
			for regexK, regexV := range defaultCacheV.(map[string]interface{}) {
				switch regexK {
				case "exclude":
					if regexV != nil {
						dc.Regex.Exclude = regexV.(string)
					}
				}
			}
		case "simplefs":
			provider := configurationtypes.CacheProvider{}
			for simplefsConfigurationK, simplefsConfigurationV := range defaultCacheV.(map[string]interface{}) {
				switch simplefsConfigurationK {
				case path:
					provider.Path, _ = simplefsConfigurationV.(string)
				case configurationPK:
					provider.Configuration = simplefsConfigurationV.(map[string]interface{})
				}
			}
			dc.SimpleFS = provider
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
		case "storers":
			if storers, ok := defaultCacheV.([]string); ok {
				dc.Storers = storers
			} else {
				if storers, ok := defaultCacheV.([]string); ok {
					dc.Storers = storers
				} else {
					for _, sv := range defaultCacheV.([]interface{}) {
						dc.Storers = append(dc.Storers, sv.(string))
					}
				}
			}
		case "default_cache_control":
			dc.DefaultCacheControl, _ = defaultCacheV.(string)
		case "max_cacheable_body_bytes":
			dc.MaxBodyBytes, _ = defaultCacheV.(uint64)
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
				if surrogate.Headers == nil {
					surrogate.Headers = make(map[string]string)
				}
				for hn, hv := range value.(map[string]interface{}) {
					surrogate.Headers[hn] = hv.(string)
				}
			case "url":
				surrogate.URL = value.(string)
			}
		}
		u[surrogateK] = surrogate
	}

	return u
}

func ParseConfiguration(baseConfiguration *middleware.BaseConfiguration, unparsedConfiguration map[string]interface{}) {
	for key, v := range unparsedConfiguration {
		switch key {
		case "api":
			baseConfiguration.API = parseAPI(v.(map[string]interface{}))
		case "cache_keys":
			baseConfiguration.CacheKeys = parseCacheKeys(v.(map[string]interface{}))
		case "default_cache":
			baseConfiguration.DefaultCache = parseDefaultCache(v.(map[string]interface{}))
		case "log_level":
			baseConfiguration.LogLevel = v.(string)
		case "urls":
			baseConfiguration.URLs = parseURLs(v.(map[string]interface{}))
		case "ykeys":
			baseConfiguration.Ykeys = parseSurrogateKeys(v.(map[string]interface{}))
		case "surrogate_keys":
			baseConfiguration.SurrogateKeys = parseSurrogateKeys(v.(map[string]interface{}))
		}
	}
}
