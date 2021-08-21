package ykeys

import (
	"net/http"
	"strings"

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
	Keys map[string]configurationtypes.YKey
}

// InitializeYKeys will initialize the ykey storage system
func InitializeYKeys(keys map[string]configurationtypes.YKey) *YKeyStorage {
	return &YKeyStorage{Keys: keys}
}

// GetValidatedTags returns the validated tags based on the key x headers
func (y *YKeyStorage) GetValidatedTags(key string, headers http.Header) []string {
	var tags []string
	return tags
}

// InvalidateTags invalidate a tag list
func (y *YKeyStorage) InvalidateTags(tags []string) []string {
	var u []string
	return u
}

// InvalidateTagURLs invalidate URLs in the stored map
func (y *YKeyStorage) InvalidateTagURLs(urls string) []string {
	u := strings.Split(urls, ",")
	return u
}

func (y *YKeyStorage) invalidateURL(url string) {}

// AddToTags add an URL to a tag list
func (y *YKeyStorage) AddToTags(url string, tags []string) {}

func (y *YKeyStorage) addToTag(url string, tag string) {}
