package providers

import (
	"fmt"
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"regexp"
	"strings"
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
)

func (s *baseStorage) ParseHeaders(value string) []string {
	r, _ := regexp.Compile(s.parent.getHeaderSeparator() + "( *)?")
	return strings.Fields(r.ReplaceAllString(value, " "))
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
		if _, found := tmp[item]; !found {
			tmp[item] = true
			list = append(list, item)
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
}

func (s *baseStorage) init(config configurationtypes.AbstractConfigurationInterface) {
	storage := make(map[string]string)
	s.Storage = storage
	s.Keys = config.GetSurrogateKeys()
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

	s.keysRegexp = keysRegexp
}

func (s *baseStorage) storeTag(tag string, cacheKey string, re *regexp.Regexp) {
	if currentValue, b := s.Storage[tag]; s.dynamic || b {
		if !re.MatchString(currentValue) {
			s.Storage[tag] = currentValue + souinStorageSeparator + cacheKey
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

func (s *baseStorage) getSurrogateControl(header http.Header) string {
	return getCandidateHeader(header, s.parent.getOrderedSurrogateControlHeadersCandidate)
}

func (s *baseStorage) getSurrogateKey(header http.Header) string {
	return getCandidateHeader(header, s.parent.getOrderedSurrogateKeyHeadersCandidate)
}

func (s *baseStorage) purgeTag(tag string) []string {
	toInvalidate := s.Storage[tag]
	delete(s.Storage, tag)
	return strings.Split(toInvalidate, souinStorageSeparator)
}

// Store will take the lead to store the cache key for each provided Surrogate-key
func (s *baseStorage) Store(request *http.Request, cacheKey string) error {
	urlRegexp, e := regexp.Compile("(^" + cacheKey + "(" + souinStorageSeparator + "|$))|(" + souinStorageSeparator + cacheKey + ")|(" + souinStorageSeparator + cacheKey + "$)")
	if e != nil {
		return fmt.Errorf("the regexp with the cache key %s cannot compile", cacheKey)
	}

	keys := s.ParseHeaders(s.parent.getSurrogateKey(request.Header))

	for _, key := range keys {
		if controls := s.ParseHeaders(s.parent.getSurrogateControl(request.Header)); len(controls) != 0 {
			for _, control := range controls {
				if s.parent.candidateStore(control) {
					s.storeTag(key, cacheKey, urlRegexp)
				}
			}
		} else {
			s.storeTag(key, cacheKey, urlRegexp)
		}
	}

	return nil
}

// Purge take the request headers as parameter, retrieve the associated cache keys for the Surrogate-Keys given.
// It returns an array which one contains the cache keys to invalidate.
func (s *baseStorage) Purge(header http.Header) (cacheKeys []string, surrogateKeys []string) {
	surrogates := s.ParseHeaders(s.parent.getSurrogateKey(header))
	toInvalidate := []string{}
	for _, su := range surrogates {
		toInvalidate = append(toInvalidate, s.purgeTag(su)...)
	}

	return uniqueTag(toInvalidate), surrogates
}
