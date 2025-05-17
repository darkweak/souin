package httpcache

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/storages/core"
)

// DefaultCache the struct
type DefaultCache struct {
	// Allowed HTTP verbs to be cached by the system.
	AllowedHTTPVerbs []string `json:"allowed_http_verbs"`
	// Allowed additional status code to be cached by the system.
	AllowedAdditionalStatusCodes []int `json:"allowed_additional_status_codes"`
	// Badger provider configuration.
	Badger configurationtypes.CacheProvider `json:"badger"`
	// The cache name to use in the Cache-Status response header.
	CacheName string                 `json:"cache_name"`
	CDN       configurationtypes.CDN `json:"cdn"`
	// The default Cache-Control header value if none set by the upstream server.
	DefaultCacheControl string `json:"default_cache_control"`
	// The maximum body size (in bytes) to be stored into cache.
	MaxBodyBytes uint64 `json:"max_cacheable_body_bytes"`
	// Redis provider configuration.
	Distributed bool `json:"distributed"`
	// Headers to add to the cache key if they are present.
	Headers []string `json:"headers"`
	// Configure the global key generation.
	Key configurationtypes.Key `json:"key"`
	// Mode defines if strict or bypass.
	Mode string `json:"mode"`
	// Olric provider configuration.
	Olric configurationtypes.CacheProvider `json:"olric"`
	// Redis provider configuration.
	Redis configurationtypes.CacheProvider `json:"redis"`
	// Etcd provider configuration.
	Etcd configurationtypes.CacheProvider `json:"etcd"`
	// Nats provider configuration.
	Nats configurationtypes.CacheProvider `json:"nats"`
	// NutsDB provider configuration.
	Nuts configurationtypes.CacheProvider `json:"nuts"`
	// Otter provider configuration.
	Otter configurationtypes.CacheProvider `json:"otter"`
	// Regex to exclude cache.
	Regex configurationtypes.Regex `json:"regex"`
	// Storage providers chaining and order.
	Storers []string `json:"storers"`
	// Time before cache or backend access timeout.
	Timeout configurationtypes.Timeout `json:"timeout"`
	// Time to live.
	TTL configurationtypes.Duration `json:"ttl"`
	// SimpleFS provider configuration.
	SimpleFS configurationtypes.CacheProvider `json:"simplefs"`
	// Stale time to live.
	Stale configurationtypes.Duration `json:"stale"`
	// Disable the coalescing system.
	DisableCoalescing bool `json:"disable_coalescing"`
}

// GetAllowedHTTPVerbs returns the allowed verbs to cache
func (d *DefaultCache) GetAllowedHTTPVerbs() []string {
	return d.AllowedHTTPVerbs
}

// GetAllowedAdditionalStatusCodes returns the allowed verbs to cache
func (d *DefaultCache) GetAllowedAdditionalStatusCodes() []int {
	return d.AllowedAdditionalStatusCodes
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

// GetMode returns mdoe configuration
func (d *DefaultCache) GetMode() string {
	return d.Mode
}

// GetNats returns nats configuration
func (d *DefaultCache) GetNats() configurationtypes.CacheProvider {
	return d.Nats
}

// GetNuts returns nuts configuration
func (d *DefaultCache) GetNuts() configurationtypes.CacheProvider {
	return d.Nuts
}

// GetOtter returns otter configuration
func (d *DefaultCache) GetOtter() configurationtypes.CacheProvider {
	return d.Otter
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

// GetSimpleFS returns simplefs configuration
func (d *DefaultCache) GetSimpleFS() configurationtypes.CacheProvider {
	return d.SimpleFS
}

// GetStorers returns the chianed storers
func (d *DefaultCache) GetStorers() []string {
	return d.Storers
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

// GetMaxBodyBytes returns the maximum body size (in bytes) to be cached
func (d *DefaultCache) GetMaxBodyBytes() uint64 {
	return d.MaxBodyBytes
}

// IsCoalescingDisable returns if the coalescing is disabled
func (d *DefaultCache) IsCoalescingDisable() bool {
	return d.DisableCoalescing
}

// Configuration holder
type Configuration struct {
	// Default cache to fallback on when none are redefined.
	DefaultCache DefaultCache
	// API endpoints enablers.
	API configurationtypes.API
	// Cache keys configuration.
	CacheKeys configurationtypes.CacheKeys `json:"cache_keys"`
	// Override the ttl depending the cases.
	URLs map[string]configurationtypes.URL
	// Logger level, fallback on caddy's one when not redefined.
	LogLevel string
	// SurrogateKeys contains the surrogate keys to use with a predefined mapping
	SurrogateKeys map[string]configurationtypes.SurrogateKeys
	logger        core.Logger
}

// GetUrls get the urls list in the configuration
func (c *Configuration) GetUrls() map[string]configurationtypes.URL {
	return c.URLs
}

// GetDefaultCache get the default cache
func (c *Configuration) GetPluginName() string {
	return "caddy"
}

// GetDefaultCache get the default cache
func (c *Configuration) GetDefaultCache() configurationtypes.DefaultCacheInterface {
	return &c.DefaultCache
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
func (c *Configuration) GetLogger() core.Logger {
	return c.logger
}

// SetLogger set the logger
func (c *Configuration) SetLogger(l core.Logger) {
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
	return c.CacheKeys
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
			if v != nil {
				val, ok := v.(string)
				if ok {
					c[k], _ = strconv.ParseBool(val)
				}
			}
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
		case "Addrs", "InitAddress":
			if s, ok := v.(string); ok {
				c[k] = []string{s}
			} else {
				c[k] = v
			}
		case "Username", "Password", "ClientName", "ClientSetInfo", "ClientTrackingOptions", "SentinelUsername", "SentinelPassword", "MasterName", "IdentitySuffix":
			c[k] = v
		case "SendToReplicas", "ShuffleInit", "ClientNoTouch", "DisableRetry", "DisableCache", "AlwaysPipelining", "AlwaysRESP2", "ForceSingleClient", "ReplicaOnly", "ClientNoEvict", "ContextTimeoutEnabled", "PoolFIFO", "ReadOnly", "RouteByLatency", "RouteRandomly", "DisableIndentity":
			c[k] = true
		case "SelectDB", "CacheSizeEachConn", "RingScaleEachConn", "ReadBufferEachConn", "WriteBufferEachConn", "BlockingPoolSize", "PipelineMultiplex", "DB", "Protocol", "MaxRetries", "PoolSize", "MinIdleConns", "MaxIdleConns", "MaxActiveConns", "MaxRedirects":
			if v == false {
				c[k] = 0
			} else if v == true {
				c[k] = 1
			} else {
				c[k], _ = strconv.Atoi(v.(string))
			}
		case "ConnWriteTimeout", "MaxFlushDelay", "MinRetryBackoff", "MaxRetryBackoff", "DialTimeout", "ReadTimeout", "WriteTimeout", "PoolTimeout", "ConnMaxIdleTime", "ConnMaxLifetime":
			c[k], _ = time.ParseDuration(v.(string))
		case "MaxVersion", "MinVersion":
			strV, _ := v.(string)
			if strings.HasPrefix(strV, "TLS") {
				strV = strings.Trim(strings.TrimPrefix(strV, "TLS"), " ")
			}

			switch strV {
			case "0x0300", "SSLv3":
				c[k] = 0x0300
			case "0x0301", "1.0":
				c[k] = 0x0301
			case "0x0302", "1.1":
				c[k] = 0x0302
			case "0x0303", "1.2":
				c[k] = 0x0303
			case "0x0304", "1.3":
				c[k] = 0x0304
			}
		case "TLSConfig":
			c[k] = parseRedisConfiguration(v.(map[string]interface{}))
		}
	}

	return c
}

func parseSimpleFSConfiguration(c map[string]interface{}) map[string]interface{} {
	for k, v := range c {
		switch k {
		case "path":
			c[k] = v
		case "size":
			if v == false {
				c[k] = 0
			} else if v == true {
				c[k] = 1
			} else {
				c[k], _ = strconv.Atoi(v.(string))
			}
		case "directory_size":
			if v == false {
				c[k] = 0
			} else if v == true {
				c[k] = 1
			} else {
				c[k], _ = strconv.Atoi(v.(string))
			}
		}
	}

	return c
}

func parseConfiguration(cfg *Configuration, h *caddyfile.Dispenser, isGlobal bool) error {
	for h.Next() {
		for nesting := h.Nesting(); h.NextBlock(nesting); {
			rootOption := h.Val()
			switch rootOption {
			case "allowed_http_verbs":
				allowed := cfg.DefaultCache.AllowedHTTPVerbs
				allowed = append(allowed, h.RemainingArgs()...)
				cfg.DefaultCache.AllowedHTTPVerbs = allowed
			case "allowed_additional_status_codes":
				allowed := cfg.DefaultCache.AllowedAdditionalStatusCodes
				additional := h.RemainingArgs()
				codes := make([]int, 0)
				for _, code := range additional {
					if c, err := strconv.Atoi(code); err == nil {
						codes = append(codes, c)
					}
				}
				allowed = append(allowed, codes...)
				cfg.DefaultCache.AllowedAdditionalStatusCodes = allowed
			case "api":
				if !isGlobal {
					return h.Err("'api' block must be global")
				}
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
							default:
								return h.Errf("unsupported debug directive: %s", directive)
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
							default:
								return h.Errf("unsupported prometheus directive: %s", directive)
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
							default:
								return h.Errf("unsupported souin directive: %s", directive)
							}
						}
					default:
						return h.Errf("unsupported api directive: %s", directive)
					}
				}
				cfg.API = apiConfiguration
			case "badger":
				provider := configurationtypes.CacheProvider{Found: true}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
						provider.Configuration = parseBadgerConfiguration(provider.Configuration.(map[string]interface{}))
					default:
						return h.Errf("unsupported badger directive: %s", directive)
					}
				}
				cfg.DefaultCache.Badger = provider
			case "cache_keys":
				CacheKeys := cfg.CacheKeys
				if CacheKeys == nil {
					CacheKeys = make(configurationtypes.CacheKeys, 0)
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
						case "disable_scheme":
							ck.DisableScheme = true
						case "disable_vary":
							ck.DisableVary = true
						case "template":
							ck.Template = h.RemainingArgs()[0]
						case "hash":
							ck.Hash = true
						case "hide":
							ck.Hide = true
						case "headers":
							ck.Headers = h.RemainingArgs()
						default:
							return h.Errf("unsupported cache_keys (%s) directive: %s", rg, directive)
						}
					}

					CacheKeys = append(CacheKeys, configurationtypes.CacheKey{configurationtypes.RegValue{Regexp: regexp.MustCompile(rg)}: ck})
				}
				cfg.CacheKeys = CacheKeys
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
						cdn.Dynamic = true
						args := h.RemainingArgs()
						if len(args) > 0 {
							cdn.Dynamic, _ = strconv.ParseBool(args[0])
						}
					case "email":
						cdn.Email = h.RemainingArgs()[0]
					case "hostname":
						cdn.Hostname = h.RemainingArgs()[0]
					case "network":
						cdn.Network = h.RemainingArgs()[0]
					case "provider":
						cdn.Provider = h.RemainingArgs()[0]
					case "service_id":
						cdn.ServiceID = h.RemainingArgs()[0]
					case "strategy":
						cdn.Strategy = h.RemainingArgs()[0]
					case "zone_id":
						cdn.ZoneID = h.RemainingArgs()[0]
					default:
						return h.Errf("unsupported cdn directive: %s", directive)
					}
				}
				cfg.DefaultCache.CDN = cdn
			case "default_cache_control":
				args := h.RemainingArgs()
				cfg.DefaultCache.DefaultCacheControl = strings.Join(args, " ")
			case "max_cacheable_body_bytes":
				args := h.RemainingArgs()
				maxBodyBytes, err := strconv.ParseUint(args[0], 10, 64)
				if err != nil {
					return h.Errf("unsupported max_cacheable_body_bytes: %s", args)
				} else {
					cfg.DefaultCache.MaxBodyBytes = maxBodyBytes
				}
			case "etcd":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{Found: true}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					default:
						return h.Errf("unsupported etcd directive: %s", directive)
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
					case "disable_scheme":
						config_key.DisableScheme = true
					case "disable_vary":
						config_key.DisableVary = true
					case "template":
						config_key.Template = h.RemainingArgs()[0]
					case "hash":
						config_key.Hash = true
					case "hide":
						config_key.Hide = true
					case "headers":
						config_key.Headers = h.RemainingArgs()
					default:
						return h.Errf("unsupported key directive: %s", directive)
					}
				}
				cfg.DefaultCache.Key = config_key
			case "log_level":
				args := h.RemainingArgs()
				cfg.LogLevel = args[0]
			case "mode":
				args := h.RemainingArgs()
				if len(args) > 1 {
					return h.Errf("mode must contains only one arg: %s given", args)
				}
				cfg.DefaultCache.Mode = args[0]
			case "nats":
				provider := configurationtypes.CacheProvider{Found: true}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "url":
						urlArgs := h.RemainingArgs()
						provider.URL = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					default:
						return h.Errf("unsupported nats directive: %s", directive)
					}
				}
				cfg.DefaultCache.Nats = provider
			case "nuts":
				provider := configurationtypes.CacheProvider{Found: true}
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
					default:
						return h.Errf("unsupported nuts directive: %s", directive)
					}
				}
				cfg.DefaultCache.Nuts = provider
			case "otter":
				provider := configurationtypes.CacheProvider{Found: true}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
					default:
						return h.Errf("unsupported otter directive: %s", directive)
					}
				}
				cfg.DefaultCache.Otter = provider
			case "olric":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{Found: true}
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
					default:
						return h.Errf("unsupported olric directive: %s", directive)
					}
				}
				cfg.DefaultCache.Olric = provider
			case "redis":
				cfg.DefaultCache.Distributed = true
				provider := configurationtypes.CacheProvider{Found: true}
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
					default:
						return h.Errf("unsupported redis directive: %s", directive)
					}
				}
				cfg.DefaultCache.Redis = provider
			case "regex":
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "exclude":
						cfg.DefaultCache.Regex.Exclude = h.RemainingArgs()[0]
					default:
						return h.Errf("unsupported regex directive: %s", directive)
					}
				}
			case "simplefs":
				provider := configurationtypes.CacheProvider{Found: true}
				for nesting := h.Nesting(); h.NextBlock(nesting); {
					directive := h.Val()
					switch directive {
					case "path":
						urlArgs := h.RemainingArgs()
						provider.Path = urlArgs[0]
					case "configuration":
						provider.Configuration = parseCaddyfileRecursively(h)
						provider.Configuration = parseSimpleFSConfiguration(provider.Configuration.(map[string]interface{}))
					default:
						return h.Errf("unsupported simplefs directive: %s", directive)
					}
				}
				cfg.DefaultCache.SimpleFS = provider
			case "stale":
				args := h.RemainingArgs()
				stale, err := time.ParseDuration(args[0])
				if err == nil {
					cfg.DefaultCache.Stale.Duration = stale
				}
			case "storers":
				args := h.RemainingArgs()
				cfg.DefaultCache.Storers = args
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
					default:
						return h.Errf("unsupported timeout directive: %s", directive)
					}
				}
				cfg.DefaultCache.Timeout = timeout
			case "ttl":
				args := h.RemainingArgs()
				ttl, err := time.ParseDuration(args[0])
				if err == nil {
					cfg.DefaultCache.TTL.Duration = ttl
				}
			case "disable_coalescing":
				cfg.DefaultCache.DisableCoalescing = true
			default:
				return h.Errf("unsupported root directive: %s", rootOption)
			}
		}
	}

	return nil
}
