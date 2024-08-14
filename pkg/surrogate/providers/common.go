package providers

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/storages/core"
)

const (
	cdnCacheControl       = "CDN-Cache-Control"
	cacheGroupKey         = "Cache-Groups"
	surrogateKey          = "Surrogate-Key"
	surrogateControl      = "Surrogate-Control"
	cacheControl          = "Cache-Control"
	noStoreDirective      = "no-store"
	souinStorageSeparator = ","
	souinCacheControl     = "Souin-Cache-Control"
	fastlyCacheControl    = "Fastly-Cache-Control"
	edgeCacheTag          = "Edge-Cache-Tag"
	cacheTags             = "Cache-Tags"
	cacheTag              = "Cache-Tag"

	surrogatePrefix = "SURROGATE_"
)

var storageToInfiniteTTLMap = map[string]time.Duration{
	"BADGER":                 types.OneYearDuration,
	"ETCD":                   types.OneYearDuration,
	"NUTS":                   0,
	"OLRIC":                  types.OneYearDuration,
	"OTTER":                  types.OneYearDuration,
	"REDIS":                  -1,
	types.DefaultStorageName: types.OneYearDuration,
}

func (s *baseStorage) ParseHeaders(value string) []string {
	return strings.Split(value, s.parent.getHeaderSeparator())
}

func getCandidateHeader(header http.Header, getCandidates func() []string) (string, string) {
	candidates := getCandidates()
	for _, candidate := range candidates {
		if h := header.Get(candidate); h != "" {
			return candidate, h
		}
	}

	return candidates[len(candidates)-1], ""
}

func isSafeHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}

	return false
}

func uniqueTag(values []string) []string {
	tmp := make(map[string]bool)
	list := []string{}

	for _, item := range values {
		if item == "" {
			continue
		}
		if _, found := tmp[item]; !found {
			tmp[item] = true
			i, _ := url.QueryUnescape(item)
			list = append(list, i)
		}
	}

	return list
}

type keysRegexpInner struct {
	Headers map[string]*regexp.Regexp
	Url     *regexp.Regexp
}

type baseStorage struct {
	parent     SurrogateInterface
	Storage    types.Storer
	Keys       map[string]configurationtypes.SurrogateKeys
	keysRegexp map[string]keysRegexpInner
	dynamic    bool
	keepStale  bool
	logger     core.Logger
	mu         *sync.Mutex
	duration   time.Duration
}

func (s *baseStorage) init(config configurationtypes.AbstractConfigurationInterface, defaultStorerName string) {
	if configuration, ok := config.GetSurrogateKeys()["_configuration"]; ok {
		storer := core.GetRegisteredStorer(configuration.SurrogateConfiguration.Storer)
		if storer == nil {
			storer = core.GetRegisteredStorer(types.DefaultStorageName + "-")
			if storer == nil {
				config.GetLogger().Errorf("Impossible to retrieve the storers %s for the surrogate-keys from it's configuration", configuration.SurrogateConfiguration.Storer)
			}
		}

		s.Storage = storer
	} else {
		storer := core.GetRegisteredStorer(defaultStorerName)
		if storer == nil {
			config.GetLogger().Errorf("Impossible to retrieve the storers %s for the surrogate-keys fallback to the default storage", configuration.SurrogateConfiguration.Storer)
			storer = core.GetRegisteredStorer(types.DefaultStorageName + "-")
			if storer == nil {
				config.GetLogger().Error("Impossible to retrieve the default storer")
			}
		}

		s.Storage = storer
	}

	s.Keys = config.GetSurrogateKeys()
	s.keepStale = config.GetDefaultCache().GetCDN().Strategy != "hard"
	keysRegexp := make(map[string]keysRegexpInner, len(s.Keys))
	baseRegexp := regexp.MustCompile(".+")

	for key, regexps := range s.Keys {
		headers := make(map[string]*regexp.Regexp, len(regexps.Headers))
		for hk, hv := range regexps.Headers {
			headers[hk] = baseRegexp
			if hv != "" {
				headers[hk] = regexp.MustCompile(hv)
			}
		}

		innerKey := keysRegexpInner{Headers: headers, Url: baseRegexp}

		if regexps.URL != "" {
			innerKey.Url = regexp.MustCompile(regexps.URL)
		}

		keysRegexp[key] = innerKey
	}

	s.dynamic = config.GetDefaultCache().GetCDN().Dynamic
	s.logger = config.GetLogger()
	s.keysRegexp = keysRegexp
	s.mu = &sync.Mutex{}
	s.duration = storageToInfiniteTTLMap[s.Storage.Name()]
}

func (s *baseStorage) storeTag(tag string, cacheKey string, re *regexp.Regexp) {
	defer s.mu.Unlock()
	s.mu.Lock()
	currentValue := string(s.Storage.Get(surrogatePrefix + tag))
	if !re.MatchString(currentValue) {
		s.logger.Debugf("Store the tag %s", tag)
		_ = s.Storage.Set(surrogatePrefix+tag, []byte(currentValue+souinStorageSeparator+cacheKey), s.duration)
	}
}

func (*baseStorage) candidateStore(tag string) bool {
	return !strings.Contains(tag, noStoreDirective)
}

func (*baseStorage) getOrderedSurrogateKeyHeadersCandidate() []string {
	return []string{
		surrogateKey,
		edgeCacheTag,
		cacheTags,
	}
}

func (*baseStorage) getOrderedSurrogateControlHeadersCandidate() []string {
	return []string{
		souinCacheControl,
		surrogateControl,
		cdnCacheControl,
		cacheControl,
	}
}

func (s *baseStorage) GetSurrogateControl(header http.Header) (string, string) {
	return getCandidateHeader(header, s.parent.getOrderedSurrogateControlHeadersCandidate)
}

func (s *baseStorage) GetSurrogateControlName() string {
	return s.parent.getOrderedSurrogateControlHeadersCandidate()[0]
}

func (s *baseStorage) getSurrogateKey(header http.Header) string {
	_, v := getCandidateHeader(header, s.parent.getOrderedSurrogateKeyHeadersCandidate)
	return v
}

func (s *baseStorage) purgeTag(tag string) []string {
	toInvalidate := string(s.Storage.Get(surrogatePrefix + tag))
	s.logger.Debugf("Purge the tag %s", tag)
	if !s.keepStale {
		s.Storage.Delete(surrogatePrefix + tag)
	}
	return strings.Split(toInvalidate, souinStorageSeparator)
}

// Store will take the lead to store the cache key for each provided Surrogate-key
func (s *baseStorage) Store(response *http.Response, cacheKey, uri, basekey string) error {
	h := response.Header

	cacheKey = url.QueryEscape(cacheKey)

	urlRegexp := regexp.MustCompile("(^|" + regexp.QuoteMeta(souinStorageSeparator) + ")" + regexp.QuoteMeta(cacheKey) + "(" + regexp.QuoteMeta(souinStorageSeparator) + "|$)")
	keys := s.ParseHeaders(s.parent.getSurrogateKey(h))

	for _, key := range keys {
		_, v := s.parent.GetSurrogateControl(h)
		if controls := s.ParseHeaders(v); len(controls) != 0 {
			if len(controls) == 1 && controls[0] == "" {
				s.storeTag(key, cacheKey, urlRegexp)

				continue
			}
			for _, control := range controls {
				if s.parent.candidateStore(control) {
					s.storeTag(key, cacheKey, urlRegexp)

					break
				}
			}
		} else {
			s.storeTag(key, cacheKey, urlRegexp)
		}
	}

	urlRegexp = regexp.MustCompile("(^|" + regexp.QuoteMeta(souinStorageSeparator) + ")" + regexp.QuoteMeta(basekey) + "(" + regexp.QuoteMeta(souinStorageSeparator) + "|$)")
	s.storeTag(uri, basekey, urlRegexp)

	return nil
}

// Purge take the request headers as parameter, retrieve the associated cache keys for the Surrogate-Key given.
// It returns an array which one contains the cache keys to invalidate.
func (s *baseStorage) Purge(header http.Header) (cacheKeys []string, surrogateKeys []string) {
	surrogates := s.ParseHeaders(s.parent.getSurrogateKey(header))
	toInvalidate := []string{}
	for _, su := range surrogates {
		toInvalidate = append(toInvalidate, s.purgeTag(su)...)
	}

	s.logger.Debugf("Purge the following tags: %+v", toInvalidate)

	return uniqueTag(toInvalidate), surrogates
}

// Invalidate the grouped responses from the Cache-Group-Invalidation HTTP response header
func (s *baseStorage) Invalidate(method string, headers http.Header) {
	if !isSafeHTTPMethod(method) {
		for _, group := range headers["Cache-Group-Invalidation"] {
			s.purgeTag(group)
		}
	}
}

// List returns the stored keys associated to resources
func (s *baseStorage) List() map[string]string {
	return s.Storage.MapKeys(surrogatePrefix)
}

// Destruct method will shutdown properly the provider
func (s *baseStorage) Destruct() error {
	return s.Storage.Reset()
}
