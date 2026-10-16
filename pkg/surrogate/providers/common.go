package providers

import (
	"bytes"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/rfc"
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
	"GO-REDIS":               types.OneYearDuration,
	"NUTS":                   0,
	"OLRIC":                  types.OneYearDuration,
	"OTTER":                  types.OneYearDuration,
	"REDIS":                  -1,
	"SIMPLEFS":               0,
	types.DefaultStorageName: types.OneYearDuration,
}

func (s *baseStorage) ParseHeaders(value string) []string {
	res := strings.Split(value, s.parent.getHeaderSeparator())
	for i, v := range res {
		res[i] = strings.TrimSpace(v)
	}

	return res
}

// structuredFieldHeaders are the cache control headers defined as RFC 8941
// Structured Fields. One that fails to parse conveys nothing and must be
// skipped entirely, letting the next candidate decide instead.
var structuredFieldHeaders = map[string]bool{
	cdnCacheControl:  true,
	surrogateControl: true,
}

func getCandidateHeader(header http.Header, getCandidates func() []string) (string, string) {
	candidates := getCandidates()
	for _, candidate := range candidates {
		h := header.Get(candidate)
		if h == "" {
			continue
		}

		if structuredFieldHeaders[candidate] && !rfc.IsValidCacheControlDictionary(h) {
			continue
		}

		return candidate, h
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

			if strings.Contains(item, "%") {
				item, _ = url.QueryUnescape(item)
			}
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
	Storage    types.Storer
	Keys       map[string]configurationtypes.SurrogateKeys
	keysRegexp map[string]keysRegexpInner
	dynamic    bool
	keepStale  bool
	logger     core.Logger
	duration   time.Duration

	mu       sync.Mutex
	writeMu  sync.Mutex
	pending  map[string]*pendingKeys
	draining bool
}

func (s *baseStorage) init(config configurationtypes.AbstractConfigurationInterface, defaultStorerName string) {
	if configuration, ok := config.GetSurrogateKeys()["_configuration"]; ok {
		storer := core.GetRegisteredStorer(configuration.Storer)
		if storer == nil {
			storer = core.GetRegisteredStorer(types.DefaultStorageName + "-")
			if storer == nil {
				config.GetLogger().Errorf("Impossible to retrieve the storers %s for the surrogate-keys from it's configuration", configuration.Storer)
			}
		}

		s.Storage = storer
	} else {
		config.GetLogger().Debugf("Try to load the storer %s as surrogate backend", defaultStorerName)
		storer := core.GetRegisteredStorer(defaultStorerName)
		if storer == nil {
			config.GetLogger().Errorf("Impossible to retrieve the storers %s for the surrogate-keys fallback to the default storage", configuration.Storer)
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
	s.duration = storageToInfiniteTTLMap[s.Storage.Name()]
}

type pendingKeys struct {
	seen  map[string]struct{}
	order []string
}

func (s *baseStorage) storeTag(tag string, cacheKey string) {
	s.mu.Lock()
	if s.pending == nil {
		s.pending = make(map[string]*pendingKeys)
	}

	keys, ok := s.pending[tag]
	if !ok {
		keys = &pendingKeys{seen: make(map[string]struct{})}
		s.pending[tag] = keys
	}

	if _, found := keys.seen[cacheKey]; !found {
		keys.seen[cacheKey] = struct{}{}
		keys.order = append(keys.order, cacheKey)
	}

	startDrain := !s.draining
	s.draining = true
	s.mu.Unlock()

	if startDrain {
		go s.drain()
	}
}

func (s *baseStorage) drain() {
	for {
		if !s.flushPending() {
			s.mu.Lock()
			if len(s.pending) == 0 {
				s.draining = false
				s.mu.Unlock()

				return
			}
			s.mu.Unlock()
		}
	}
}

func (s *baseStorage) flushPending() bool {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.Lock()
	batch := s.pending
	s.pending = nil
	s.mu.Unlock()

	if len(batch) == 0 {
		return false
	}

	setStorer, isSetStorer := s.Storage.(core.SetStorer)
	for tag, keys := range batch {
		s.logger.Debugf("Store the tag %s", tag)

		// Native sets avoid rereading and rewriting the whole tag value.
		if isSetStorer {
			_ = setStorer.AddToSet(surrogatePrefix+tag, keys.order, s.duration)

			continue
		}

		s.appendToTag(tag, keys)
	}

	return true
}

func (s *baseStorage) appendToTag(tag string, keys *pendingKeys) {
	current := s.Storage.Get(surrogatePrefix + tag)

	for rest := current; len(rest) > 0; {
		member := rest
		if i := bytes.IndexByte(rest, souinStorageSeparator[0]); i >= 0 {
			member, rest = rest[:i], rest[i+1:]
		} else {
			rest = nil
		}

		delete(keys.seen, string(member))
	}

	if len(keys.seen) == 0 {
		return
	}

	size := len(current)
	for _, key := range keys.order {
		size += len(key) + len(souinStorageSeparator)
	}

	value := make([]byte, 0, size)
	value = append(value, current...)
	for _, key := range keys.order {
		if _, missing := keys.seen[key]; !missing {
			continue
		}

		if len(value) > 0 {
			value = append(value, souinStorageSeparator...)
		}
		value = append(value, key...)
	}

	_ = s.Storage.Set(surrogatePrefix+tag, value, s.duration)
}

func (*baseStorage) candidateStore(tag string) bool {
	return !strings.Contains(tag, noStoreDirective)
}

func (*baseStorage) getOrderedSurrogateKeyHeadersCandidate() []string {
	return []string{
		cacheGroupKey,
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
	var toInvalidate []string
	if setStorer, ok := s.Storage.(core.SetStorer); ok {
		toInvalidate = setStorer.GetSet(surrogatePrefix + tag)
	} else {
		toInvalidate = strings.Split(string(s.Storage.Get(surrogatePrefix+tag)), souinStorageSeparator)
	}

	s.logger.Debugf("Purge the tag %s", tag)

	if !s.keepStale {
		s.Storage.Delete(surrogatePrefix + tag)
	}

	return toInvalidate
}

// Store will take the lead to store the cache key for each provided Surrogate-key
func (s *baseStorage) Store(response *http.Response, cacheKey, uri string) error {
	h := response.Header

	cacheKey = url.QueryEscape(cacheKey)

	keys := s.ParseHeaders(s.parent.getSurrogateKey(h))

	for _, key := range keys {
		_, v := s.parent.GetSurrogateControl(h)
		if controls := s.ParseHeaders(v); len(controls) != 0 {
			if len(controls) == 1 && controls[0] == "" {
				s.storeTag(key, cacheKey)
				s.storeTag(uri, cacheKey)

				continue
			}
			for _, control := range controls {
				if s.parent.candidateStore(control) {
					s.storeTag(key, cacheKey)
					s.storeTag(uri, cacheKey)

					break
				}
			}
		} else {
			s.storeTag(key, cacheKey)
			s.storeTag(uri, cacheKey)
		}
	}

	return nil
}

// Purge take the request headers as parameter, retrieve the associated cache keys for the Surrogate-Key given.
// It returns an array which one contains the cache keys to invalidate.
func (s *baseStorage) Purge(header http.Header) (cacheKeys []string, surrogateKeys []string) {
	surrogates := s.ParseHeaders(s.parent.getSurrogateKey(header))
	s.flushPending()
	toInvalidate := []string{}
	for _, su := range surrogates {
		toInvalidate = append(toInvalidate, s.purgeTag(su)...)
	}

	s.logger.Debugf("Purge the following tags: %+v", toInvalidate)

	return uniqueTag(toInvalidate), surrogates
}

// Invalidate the grouped responses from the Cache-Group-Invalidation HTTP response header
func (s *baseStorage) Invalidate(method string, headers http.Header) {
	if groups := headers["Cache-Group-Invalidation"]; !isSafeHTTPMethod(method) && len(groups) > 0 {
		s.flushPending()
		for _, group := range groups {
			s.purgeTag(group)
		}
	}
}

// List returns the stored keys associated to resources
func (s *baseStorage) List() map[string]string {
	s.flushPending()

	if setStorer, ok := s.Storage.(core.SetStorer); ok {
		keys := map[string]string{}

		_ = setStorer.WalkSets(surrogatePrefix, func(key string, members []string) bool {
			keys[key] = strings.Join(members, souinStorageSeparator)

			return true
		})

		return keys
	}

	return s.Storage.MapKeys(surrogatePrefix)
}

// Destruct method will shutdown properly the provider
func (s *baseStorage) Destruct() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.Lock()
	s.pending = nil
	s.mu.Unlock()

	return s.Storage.Reset()
}
