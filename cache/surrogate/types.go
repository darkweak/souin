package surrogate

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/dgraph-io/ristretto"
)

const (
	SurrogateKeys    = "Surrogate-Keys"
	SurrogateControl = "Surrogate-Control"
	NoStoreDirective = "no-store"
)

// SurrogateStorage is the layer for Surrogate-key support storage
type SurrogateStorage struct {
	*ristretto.Cache
	Keys    map[string]configurationtypes.YKey
	dynamic bool
}

func (s *SurrogateStorage) storeTag(tag string, cacheKey string, re *regexp.Regexp) string {
	if currentValue, b := s.Get(tag); b {
		if !re.MatchString(currentValue.(string)) {
			s.Set(tag, currentValue.(string)+","+cacheKey, 1)
			return ";stored"
		}

		return ";bypass;detail=PRESENT"
	} else if s.dynamic {
		s.Set(tag, cacheKey, 1)
		return ";stored"
	}

	return ";bypass;NOT_ALLOWED"
}

func candidateStore(tag string) bool {
	return !strings.Contains(tag, NoStoreDirective)
}

// Store will take the lead to store the cache key for each provided Surrogate-key
func (s *SurrogateStorage) Store(header *http.Header, cacheKey string) error {
	urlRegexp, e := regexp.Compile("(^" + cacheKey + "(,|$))|(," + cacheKey + ")|(," + cacheKey + "$)")
	if e != nil {
		return fmt.Errorf("the regexp with the cache key %s cannot compile", cacheKey)
	}

	keys := ParseHeaders(header.Get(SurrogateKeys))
	for i, key := range keys {
		directive := ""
		if controls := ParseHeaders(header.Get(SurrogateControl)); len(controls) != 0 {
			for _, control := range controls {
				if candidateStore(control) {
					directive = s.storeTag(key, cacheKey, urlRegexp)
				}
			}
		} else {
			directive = s.storeTag(key, cacheKey, urlRegexp)
		}

		keys[i] = key + directive
	}

	header.Set(SurrogateKeys, strings.Join(keys[:], ", "))

	return nil
}

func (s *SurrogateStorage) purgeTag(tag string) {
	s.Del(tag)
}

// Purge take the request headers as parameter, retrieve the associated cache keys for the Surrogate-Keys given.
// It returns an array which one contains the cache keys to invalidate.
func (s *SurrogateStorage) Purge(header http.Header) []string {
	surrogates := ParseHeaders(header.Get(SurrogateKeys))
	for _, su := range surrogates {
		s.purgeTag(su)
	}

	return []string{}
}
