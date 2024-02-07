package providers

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/darkweak/souin/configurationtypes"
	"go.uber.org/zap"
)

const (
	cdnCacheControl       = "CDN-Cache-Control"
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

	stalePrefix = "STALE_"
)

func (s *baseStorage) ParseHeaders(value string) []string {
	return regexp.MustCompile(s.parent.getHeaderSeparator()+" *").Split(value, -1)
}

func getCandidateHeader(header http.Header, getCandidates func() []string) string {
	for _, candidate := range getCandidates() {
		if h := header.Get(candidate); h != "" {
			return h
		}
	}

	return ""
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
	Storage    *sync.Map
	Keys       map[string]configurationtypes.SurrogateKeys
	keysRegexp map[string]keysRegexpInner
	dynamic    bool
	keepStale  bool
	logger     *zap.Logger
	mu         *sync.Mutex
}

func (s *baseStorage) init(config configurationtypes.AbstractConfigurationInterface) {
	s.Storage = &sync.Map{}
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
}

func (s *baseStorage) storeTag(tag string, cacheKey string, re *regexp.Regexp) {
	defer s.mu.Unlock()
	s.mu.Lock()
	currentValue, b := s.Storage.Load(tag)
	if currentValue == nil {
		currentValue = ""
		b = s.dynamic
	}
	if s.dynamic || b {
		if !re.MatchString(currentValue.(string)) {
			s.logger.Sugar().Debugf("Store the tag %s", tag)
			s.Storage.Store(tag, currentValue.(string)+souinStorageSeparator+cacheKey)
		}
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

func (s *baseStorage) GetSurrogateControl(header http.Header) string {
	return getCandidateHeader(header, s.parent.getOrderedSurrogateControlHeadersCandidate)
}

func (s *baseStorage) getSurrogateKey(header http.Header) string {
	return getCandidateHeader(header, s.parent.getOrderedSurrogateKeyHeadersCandidate)
}

func (s *baseStorage) purgeTag(tag string) []string {
	toInvalidate, _ := s.Storage.Load(tag)
	if toInvalidate == nil {
		toInvalidate = ""
	}
	s.logger.Sugar().Debugf("Purge the tag %s", tag)
	s.Storage.Delete(tag)
	if !s.keepStale {
		stale, _ := s.Storage.Load(stalePrefix + tag)
		if stale == nil {
			stale = ""
		}
		toInvalidate = toInvalidate.(string) + "," + stale.(string)
		s.logger.Sugar().Debugf("Purge the tag %s", stalePrefix+tag)
		s.Storage.Delete(stalePrefix + tag)
	}
	return strings.Split(toInvalidate.(string), souinStorageSeparator)
}

// Store will take the lead to store the cache key for each provided Surrogate-key
func (s *baseStorage) Store(response *http.Response, cacheKey string) error {
	h := response.Header

	cacheKey = url.QueryEscape(cacheKey)
	staleKey := stalePrefix + cacheKey

	urlRegexp := regexp.MustCompile("(^|" + regexp.QuoteMeta(souinStorageSeparator) + ")" + regexp.QuoteMeta(cacheKey) + "(" + regexp.QuoteMeta(souinStorageSeparator) + "|$)")
	staleUrlRegexp := regexp.MustCompile("(^|" + regexp.QuoteMeta(souinStorageSeparator) + ")" + regexp.QuoteMeta(staleKey) + "(" + regexp.QuoteMeta(souinStorageSeparator) + "|$)")

	keys := s.ParseHeaders(s.parent.getSurrogateKey(h))

	for _, key := range keys {
		if controls := s.ParseHeaders(s.parent.GetSurrogateControl(h)); len(controls) != 0 {
			if len(controls) == 1 && controls[0] == "" {
				s.storeTag(key, cacheKey, urlRegexp)
				s.storeTag(stalePrefix+key, staleKey, staleUrlRegexp)

				continue
			}
			for _, control := range controls {
				if s.parent.candidateStore(control) {
					s.storeTag(key, cacheKey, urlRegexp)
					s.storeTag(stalePrefix+key, staleKey, staleUrlRegexp)
				}
			}
		} else {
			s.storeTag(key, cacheKey, urlRegexp)
			s.storeTag(stalePrefix+key, staleKey, staleUrlRegexp)
		}
	}

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

	return uniqueTag(toInvalidate), surrogates
}

// List returns the stored keys associated to resources
func (s *baseStorage) List() map[string]string {
	m := make(map[string]string)
	s.Storage.Range(func(key, value interface{}) bool {
		m[key.(string)] = value.(string)
		return true
	})
	return m
}

// Destruct method will shutdown properly the provider
func (s *baseStorage) Destruct() error {
	s.Storage.Range(func(key, value interface{}) bool {
		s.Storage.Delete(key)
		return true
	})

	return nil
}
