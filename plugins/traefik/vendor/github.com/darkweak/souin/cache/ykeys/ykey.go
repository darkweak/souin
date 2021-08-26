package ykeys

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/darkweak/souin/configurationtypes"
)

// The YKey system is like the Varnish one. You can invalidate cache from ykey based instead of the regexp or the plain
// URL to invalidate. It will target the referred URLs to this tag
// e.g.
// Given YKey data as
// |---------------|-----------------------------------------------------------------------------------|
// | YKey          | URLs                                                                              |
// |---------------|-----------------------------------------------------------------------------------|
// | GROUP_KEY_ONE | http://domain.com/,http://domain.com/1,http://domain.com/2,http://domain.com/4    |
// | GROUP_KEY_TWO | http://domain.com/1,http://domain.com/2,http://domain.com/3,http://domain.com/xyz |
// |---------------|-----------------------------------------------------------------------------------|
// When I send a purge request to /ykey/GROUP_KEY_ONE
// Then the cache will be purged for the list
// * http://domain.com/
// * http://domain.com/1
// * http://domain.com/2
// * http://domain.com/4
// And the data in the YKey table storage will contain
// |---------------|-------------------------------------------|
// | YKey          | URLs                                      |
// |---------------|-------------------------------------------|
// | GROUP_KEY_ONE |                                           |
// | GROUP_KEY_TWO | http://domain.com/3,http://domain.com/xyz |
// |---------------|-------------------------------------------|

// YKeyStorage is the layer for YKey support storage
type YKeyStorage struct {
	*cache.Cache
	Keys map[string]configurationtypes.YKey
}

// InitializeYKeys will initialize the ykey storage system
func InitializeYKeys(keys map[string]configurationtypes.YKey) *YKeyStorage {
	c := cache.New(1*time.Second, 2*time.Second)
	return &YKeyStorage{Cache: c, Keys: keys}
}

// GetValidatedTags returns the validated tags based on the key x headers
func (y *YKeyStorage) GetValidatedTags(key string, headers http.Header) []string {
	var tags []string
	for k, v := range y.Keys {
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
func (y *YKeyStorage) InvalidateTags(tags []string) []string {
	var u []string
	for _, tag := range tags {
		if v, e := y.Cache.Get(tag); e {
			u = append(u, y.InvalidateTagURLs(v.(string))...)
		}
	}

	return u
}

// InvalidateTagURLs invalidate URLs in the stored map
func (y *YKeyStorage) InvalidateTagURLs(urls string) []string {
	u := strings.Split(urls, ",")
	for _, url := range u {
		y.invalidateURL(url)
	}

	return u
}

func (y *YKeyStorage) invalidateURL(url string) {
	urlRegexp := regexp.MustCompile(fmt.Sprintf("(%s,)|(,%s$)|(^%s$)", url, url, url))
	for key := range y.Keys {
		v, _ := y.Cache.Get(key)
		if urlRegexp.MatchString(v.(string)) {
			y.Set(key, urlRegexp.ReplaceAllString(v.(string), ""), 1)
		}
	}
}

// AddToTags add an URL to a tag list
func (y *YKeyStorage) AddToTags(url string, tags []string) {
	for _, tag := range tags {
		y.addToTag(url, tag)
	}
}

func (y *YKeyStorage) addToTag(url string, tag string) {
	if v, e := y.Cache.Get(tag); e {
		urlRegexp := regexp.MustCompile(url)
		tmpStr := v.(string)
		if !urlRegexp.MatchString(tmpStr) {
			if tmpStr != "" {
				tmpStr += ","
			}
			y.Cache.Set(tag, tmpStr+url, 1)
		}
	}
}
