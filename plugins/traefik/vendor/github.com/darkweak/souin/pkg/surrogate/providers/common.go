package providers

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/darkweak/souin/configurationtypes"
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
	Storage    map[string]string
	Keys       map[string]configurationtypes.SurrogateKeys
	keysRegexp map[string]keysRegexpInner
	dynamic    bool
	keepStale  bool
	mu         *sync.Mutex
}

func (s *baseStorage) init(config configurationtypes.AbstractConfigurationInterface) {
	storage := make(map[string]string)
	s.Storage = storage
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
	s.keysRegexp = keysRegexp
	s.mu = &sync.Mutex{}
}

func (s *baseStorage) storeTag(tag string, cacheKey string, re *regexp.Regexp) {
	if currentValue, b := s.Storage[tag]; s.dynamic || b {
		if !re.MatchString(currentValue) {
			s.mu.Lock()
			fmt.Printf("Store the tag %s", tag)
			s.Storage[tag] = currentValue + souinStorageSeparator + cacheKey
			s.mu.Unlock()
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
	for _, candidate := range s.parent.getOrderedSurrogateControlHeadersCandidate() {
		if h := header.Get(candidate); h != "" {
			return h
		}
	}

	return ""
}

func (s *baseStorage) getSurrogateKey(header http.Header) string {
	for _, candidate := range s.parent.getOrderedSurrogateKeyHeadersCandidate() {
		if h := header.Get(candidate); h != "" {
			return h
		}
	}

	return ""
}

func (s *baseStorage) purgeTag(tag string) []string {
	toInvalidate := s.Storage[tag]
	fmt.Printf("Purge the tag %s", tag)
	s.mu.Lock()
	delete(s.Storage, tag)
	s.mu.Unlock()
	if !s.keepStale {
		toInvalidate = toInvalidate + "," + s.Storage[stalePrefix+tag]
		s.mu.Lock()
		fmt.Printf("Purge the tag %s", stalePrefix+tag)
		delete(s.Storage, stalePrefix+tag)
		s.mu.Unlock()
	}
	return strings.Split(toInvalidate, souinStorageSeparator)
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
	return s.Storage
}

// Destruct method will shutdown properly the provider
func (s *baseStorage) Destruct() error {
	s.Storage = make(map[string]string)

	return nil
}
