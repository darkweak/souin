package httpcache

import (
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

// DefaultCache the struct
type DefaultCache struct {
	// Allowed HTTP verbs to be cached by the system.
	AllowedHTTPVerbs []string `json:"allowed_http_verbs"`
	// Badger provider configuration.
	Badger configurationtypes.CacheProvider `json:"badger"`
	// The cache name to use in the Cache-Status response header.
	CacheName string                 `json:"cache_name"`
	CDN       configurationtypes.CDN `json:"cdn"`
	// The default Cache-Control header value if none set by the upstream server.
	DefaultCacheControl string `json:"default_cache_control"`
	// Redis provider configuration.
	Distributed bool `json:"distributed"`
	// Headers to add to the cache key if they are present.
	Headers []string `json:"headers"`
	// Configure the global key generation.
	Key configurationtypes.Key `json:"key"`
	// Olric provider configuration.
	Olric configurationtypes.CacheProvider `json:"olric"`
	// Redis provider configuration.
	Redis configurationtypes.CacheProvider `json:"redis"`
	// Etcd provider configuration.
	Etcd configurationtypes.CacheProvider `json:"etcd"`
	// NutsDB provider configuration.
	Nuts configurationtypes.CacheProvider `json:"nuts"`
	// Regex to exclude cache.
	Regex configurationtypes.Regex `json:"regex"`
	// Time before cache or backend access timeout.
	Timeout configurationtypes.Timeout `json:"timeout"`
	// Time to live.
	TTL configurationtypes.Duration `json:"ttl"`
	// Stale time to live.
	Stale configurationtypes.Duration `json:"stale"`
}

// GetAllowedHTTPVerbs returns the allowed verbs to cache
func (d *DefaultCache) GetAllowedHTTPVerbs() []string {
	return d.AllowedHTTPVerbs
}

// GetBadger returns the Badger configuration
func (d *DefaultCache) GetBadger() configurationtypes.CacheProvider {
	return d.Badger
}

// GetCacheName returns the cache name to use in the Cache-Status response header
func (d *DefaultCache) GetCacheName() string {
	return d.CacheName
}

// GetCDN returns the CDN configuration
func (d *DefaultCache) GetCDN() configurationtypes.CDN {
	return d.CDN
}

// GetDistributed returns if it uses Olric or not as provider
func (d *DefaultCache) GetDistributed() bool {
	return d.Distributed
}

// GetHeaders returns the default headers that should be cached
func (d *DefaultCache) GetHeaders() []string {
	return d.Headers
}

// GetKey returns the default Key generation strategy
func (d *DefaultCache) GetKey() configurationtypes.Key {
	return d.Key
}

// GetEtcd returns etcd configuration
func (d *DefaultCache) GetEtcd() configurationtypes.CacheProvider {
	return d.Etcd
}

// GetNuts returns nuts configuration
func (d *DefaultCache) GetNuts() configurationtypes.CacheProvider {
	return d.Nuts
}

// GetOlric returns olric configuration
func (d *DefaultCache) GetOlric() configurationtypes.CacheProvider {
	return d.Olric
}

// GetRedis returns redis configuration
func (d *DefaultCache) GetRedis() configurationtypes.CacheProvider {
	return d.Redis
}

// GetRegex returns the regex that shouldn't be cached
func (d *DefaultCache) GetRegex() configurationtypes.Regex {
	return d.Regex
}

// GetTimeout returns the backend and cache timeouts
func (d *DefaultCache) GetTimeout() configurationtypes.Timeout {
	return d.Timeout
}

// GetTTL returns the default TTL
func (d *DefaultCache) GetTTL() time.Duration {
	return d.TTL.Duration
}

// GetStale returns the stale duration
func (d *DefaultCache) GetStale() time.Duration {
	return d.Stale.Duration
}

// GetDefaultCacheControl returns the configured default cache control value
func (d *DefaultCache) GetDefaultCacheControl() string {
	return d.DefaultCacheControl
}

// Configuration holder
type Configuration struct {
	// Default cache to fallback on when none are redefined.
	DefaultCache *DefaultCache
	// API endpoints enablers.
	API configurationtypes.API
	// Cache keys configuration.
	CfgCacheKeys []map[string]configurationtypes.Key
	// Override the ttl depending the cases.
	URLs map[string]configurationtypes.URL
	// Logger level, fallback on caddy's one when not redefined.
	LogLevel string
	// SurrogateKeys contains the surrogate keys to use with a predefined mapping
	SurrogateKeys map[string]configurationtypes.SurrogateKeys
	cacheKeys     configurationtypes.CacheKeys
	logger        *zap.Logger
}

// GetUrls get the urls list in the configuration
func (c *Configuration) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetDefaultCache get the default cache
func (c *Configuration) GetDefaultCache() configurationtypes.DefaultCacheInterface {
	return c.DefaultCache
}

// GetAPI get the default cache
func (c *Configuration) GetAPI() configurationtypes.API {
	return c.API
}

// GetLogLevel get the log level
func (c *Configuration) GetLogLevel() string {
	return c.LogLevel
}

// GetLogger get the logger
func (c *Configuration) GetLogger() *zap.Logger {
	return c.logger
}

// SetLogger set the logger
func (c *Configuration) SetLogger(l *zap.Logger) {
	c.logger = l
}

// GetYkeys get the ykeys list
func (c *Configuration) GetYkeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}

// GetSurrogateKeys get the surrogate keys list
func (c *Configuration) GetSurrogateKeys() map[string]configurationtypes.SurrogateKeys {
	return nil
}

// GetCacheKeys get the cache keys rules to override
func (c *Configuration) GetCacheKeys() configurationtypes.CacheKeys {
	return c.cacheKeys
}

var _ configurationtypes.AbstractConfigurationInterface = (*Configuration)(nil)

func parseCaddyfileRecursively(h *caddyfile.Dispenser) interface{} {
	input := make(map[string]interface{})
	for nesting := h.Nesting(); h.NextBlock(nesting); {
		val := h.Val()
		if val == "}" || val == "{" {
			continue
		}
		args := h.RemainingArgs()
		if len(args) == 1 {
			input[val] = args[0]
		} else if len(args) > 1 {
			input[val] = args
		} else {
			input[val] = parseCaddyfileRecursively(h)
		}
	}

	return input
}

func parseBadgerConfiguration(c map[string]interface{}) map[string]interface{} {
	for k, v := range c {
		switch k {
		case "Dir", "ValueDir":
			c[k] = v
		case "SyncWrites", "ReadOnly", "InMemory", "MetricsEnabled", "CompactL0OnClose", "LmaxCompaction", "VerifyValueChecksum", "BypassLockGuard", "DetectConflicts":
			c[k] = true
		case "NumVersionsToKeep", "NumGoroutines", "MemTableSize", "BaseTableSize", "BaseLevelSize", "LevelSizeMultiplier", "TableSizeMultiplier", "MaxLevels", "ValueThreshold", "NumMemtables", "BlockSize", "BlockCacheSize", "IndexCacheSize", "NumLevelZeroTables", "NumLevelZeroTablesStall", "ValueLogFileSize", "NumCompactors", "ZSTDCompressionLevel", "ChecksumVerificationMode", "NamespaceOffset":
			c[k], _ = strconv.Atoi(v.(string))
		case "Compression", "ValueLogMaxEntries":
			c[k], _ = strconv.ParseUint(v.(string), 10, 32)
		case "VLogPercentile", "BloomFalsePositive":
			c[k], _ = strconv.ParseFloat(v.(string), 64)
		case "EncryptionKey":
			c[k] = []byte(v.(string))
		case "EncryptionKeyRotationDuration":
			c[k], _ = time.ParseDuration(v.(string))
		}
	}

	return c
}

func parseRedisConfiguration(c map[string]interface{}) map[string]interface{} {
	for k, v := range c {
		switch k {
		case "Network", "Addr", "Username", "Password":
			c[k] = v
		case "PoolFIFO":
			c[k] = true
		case "DB", "MaxRetries", "PoolSize", "MinIdleConns", "MaxIdleConns":
			c[k], _ = strconv.Atoi(v.(string))
		case "MinRetryBackoff", "MaxRetryBackoff", "DialTimeout", "ReadTimeout", "WriteTimeout", "PoolTimeout", "ConnMaxIdleTime", "ConnMaxLifetime":
			c[k], _ = time.ParseDuration(v.(string))
		}
	}

	return c
}

func parseConfiguration(cfg *Configuration, h *caddyfile.Dispenser, isBlocking bool) error {
	for h.Next() {
		for nesting := h.Nesting(); h.NextBlock(nesting); {
			rootOption := h.Val()
			switch rootOption {
			case "allowed_http_verbs":
				allowed := cfg.DefaultCache.AllowedHTTPVerbs
				allowed = append(allowed, h.RemainingArgs()...)
				cfg.DefaultCache.AllowedHTTPVerbs = allowed
			case "api":
				apiConfiguration := configurationtypes.API{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "basepath":
						apiConfiguration.BasePath = h.RemainingArgs()[0]
					case "debug":
						apiConfiguration.Debug = configurationtypes.APIEndpoint{}
						apiConfiguration.Debug.Enable = true
						for nesting := h.Nesting(); h.NextBlock(nesting); {
							directive := h.Val()
							switch directive {
							case "basepath":
								apiConfiguration.Debug.BasePath = h.RemainingArgs()[0]
							}
						}
					case "prometheus":
						apiConfiguration.Prometheus = configurationtypes.APIEndpoint{}
						apiConfiguration.Prometheus.Enable = true
						for nesting := h.Nesting(); h.NextBlock(nesting); {
							directive := h.Val()
							switch directive {
							case "basepath":
								apiConfiguration.Prometheus.BasePath = h.RemainingArgs()[0]
							}
						}
					case "souin":
						apiConfiguration.Souin = configurationtypes.APIEndpoint{}
						apiConfiguration.Souin.Enable = true
						for nesting := h.Nesting(); h.NextBlock(nesting); {
							directive := h.Val()
							switch directive {
							case "basepath":
								apiConfiguration.Souin.BasePath = h.RemainingArgs()[0]
							}
						}
					}
				}
				cfg.API = apiConfiguration
			case "badger":
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
						provider.Configuration = parseBadgerConfiguration(provider.Configuration.(map[string]interface{}))
					}
				}
				cfg.DefaultCache.Badger = provider
			case "cache_keys":
				cacheKeys := cfg.CfgCacheKeys
				if cacheKeys == nil {
					cacheKeys = make([]map[string]configurationtypes.Key, 0)
				}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					rg := h.Val()
					ck := configurationtypes.Key{}

					for nesting := h.Nesting(); h.NextBlock(nesting); {
						directive := h.Val()
						switch directive {
						case "disable_body":
							ck.DisableBody = true
						case "disable_host":
							ck.DisableHost = true
						case "disable_method":
							ck.DisableMethod = true
						case "disable_query":
							ck.DisableQuery = true
						case "hide":
							ck.Hide = true
						case "headers":
							ck.Headers = h.RemainingArgs()
						}
					}

					cacheKeys = append(cacheKeys, map[string]configurationtypes.Key{rg: ck})
				}
				cfg.CfgCacheKeys = cacheKeys
			case "cache_name":
				args := h.RemainingArgs()
				cfg.DefaultCache.CacheName = args[0]
			case "cdn":
				cdn := configurationtypes.CDN{
					Dynamic: true,
				}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "api_key":
						cdn.APIKey = h.RemainingArgs()[0]
					case "dynamic":
						if len(h.RemainingArgs()) > 0 {
							cdn.Dynamic, _ = strconv.ParseBool(h.RemainingArgs()[0])
						}
					case "hostname":
						cdn.Hostname = h.RemainingArgs()[0]
					case "network":
						cdn.Network = h.RemainingArgs()[0]
					case "provider":
						cdn.Provider = h.RemainingArgs()[0]
					case "strategy":
						cdn.Strategy = h.RemainingArgs()[0]
					}
				}
				cfg.DefaultCache.CDN = cdn
			case "default_cache_control":
				args := h.RemainingArgs()
				cfg.DefaultCache.DefaultCacheControl = strings.Join(args, " ")
			case "etcd":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					}
				}
				cfg.DefaultCache.Etcd = provider
			case "headers":
				cfg.DefaultCache.Headers = append(cfg.DefaultCache.Headers, h.RemainingArgs()...)
			case "key":
				config_key := configurationtypes.Key{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "disable_body":
						config_key.DisableBody = true
					case "disable_host":
						config_key.DisableHost = true
					case "disable_method":
						config_key.DisableMethod = true
					case "disable_query":
						config_key.DisableQuery = true
					case "hide":
						config_key.Hide = true
					case "headers":
						config_key.Headers = h.RemainingArgs()
					}
				}
				cfg.DefaultCache.Key = config_key
			case "log_level":
				args := h.RemainingArgs()
				cfg.LogLevel = args[0]
			case "nuts":
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "url":
						urlArgs := h.RemainingArgs()
						provider.URL = urlArgs[0]
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					}
				}
				cfg.DefaultCache.Nuts = provider
			case "olric":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "url":
						urlArgs := h.RemainingArgs()
						provider.URL = urlArgs[0]
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					}
				}
				cfg.DefaultCache.Olric = provider
			case "redis":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "url":
						urlArgs := h.RemainingArgs()
						provider.URL = urlArgs[0]
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
						provider.Configuration = parseRedisConfiguration(provider.Configuration.(map[string]interface{}))
					}
				}
				cfg.DefaultCache.Redis = provider
			case "regex":
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "exclude":
						cfg.DefaultCache.Regex.Exclude = h.RemainingArgs()[0]
					}
				}
			case "stale":
				args := h.RemainingArgs()
				stale, err := time.ParseDuration(args[0])
				if err == nil {
					cfg.DefaultCache.Stale.Duration = stale
				}
			case "timeout":
				timeout := configurationtypes.Timeout{}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "backend":
						d := configurationtypes.Duration{}
						ttl, err := time.ParseDuration(h.RemainingArgs()[0])
						if err == nil {
							d.Duration = ttl
						}
						timeout.Backend = d
					case "cache":
						d := configurationtypes.Duration{}
						ttl, err := time.ParseDuration(h.RemainingArgs()[0])
						if err == nil {
							d.Duration = ttl
						}
						timeout.Cache = d
					}
				}
				cfg.DefaultCache.Timeout = timeout
			case "ttl":
				args := h.RemainingArgs()
				ttl, err := time.ParseDuration(args[0])
				if err == nil {
					cfg.DefaultCache.TTL.Duration = ttl
				}
			default:
				if isBlocking {
					return h.Errf("unsupported root directive: %s", rootOption)
				}
			}
		}
	}

	return nil
}
