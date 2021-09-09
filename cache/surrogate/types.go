package surrogate

import (
	"fmt"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/dgraph-io/ristretto"
	"net/http"
	"regexp"
	"strings"
)

// SurrogateStorage is the layer for Surrogate-key support storage
type SurrogateStorage struct {
	*ristretto.Cache
	Keys    map[string]configurationtypes.YKey
	dynamic bool
}

// GetValidatedTags returns the validated tags based on the key x headers
func (s *SurrogateStorage) GetValidatedTags(key string, headers http.Header) []string {
	var tags []string
	for k, v := range s.Keys {
		valid := true
		if v.URL != "" {
			if r, e := regexp.MatchString(v.URL, key); !r || e != nil {
				continue
			}
		}
		if v.Headers != nil {
			for h, hValue := range v.Headers {
				if res, err := regexp.MatchString(hValue, headers.Get(h)); !res || err != nil {
					valid = false
					break
				}
			}
		}
		if valid {
			tags = append(tags, k)
		}
	}

	return tags
}

// InvalidateTags invalidate a tag list
func (s *SurrogateStorage) InvalidateTags(tags []string) []string {
	var u []string
	for _, tag := range tags {
		if v, e := s.Cache.Get(tag); e {
			u = append(u, s.InvalidateTagURLs(v.(string))...)
		}
	}

	return u
}

// InvalidateTagURLs invalidate URLs in the stored map
func (s *SurrogateStorage) InvalidateTagURLs(urls string) []string {
	u := strings.Split(urls, ",")
	for _, url := range u {
		s.invalidateURL(url)
	}
	return u
}

func (s *SurrogateStorage) invalidateURL(url string) {
	urlRegexp := regexp.MustCompile(fmt.Sprintf("(%s,)|(,%s$)|(^%s$)", url, url, url))
	for key := range s.Keys {
		v, found := s.Cache.Get(key)
		if found && urlRegexp.MatchString(v.(string)) {
			s.Set(key, urlRegexp.ReplaceAllString(v.(string), ""), 1)
		}
	}
}

// AddToTags add an URL to a tag list
func (s *SurrogateStorage) AddToTags(url string, tags []string) {
	for _, tag := range tags {
		s.addToTag(url, tag)
	}
}

func (s *SurrogateStorage) addToTag(url string, tag string) {
	if v, e := s.Cache.Get(tag); e {
		urlRegexp := regexp.MustCompile(url)
		tmpStr := v.(string)
		if !urlRegexp.MatchString(tmpStr) {
			if tmpStr != "" {
				tmpStr += ","
			}
			s.Cache.Set(tag, tmpStr+url, 1)
		}
	}
}
