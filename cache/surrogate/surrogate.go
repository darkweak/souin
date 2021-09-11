package surrogate

// The Surrogate key system is like the ykey one in Varnish and the same as Fastly.
// You can send a PURGE request on the Souin API endpoint with the Surrogate header to invalidate the filled keys.
// The Surrogate header CAN contains one or multiple keys separated by a comma as mentioned in the RFC.
// e.g.
// Given the Surrogate-Keys data as
// |---------------|-----------------------------------------------------------------------------------|
// | YKey          | URLs                                                                              |
// |---------------|-----------------------------------------------------------------------------------|
// | GROUP_KEY_ONE | http://domain.com/,http://domain.com/1,http://domain.com/2,http://domain.com/4    |
// | GROUP_KEY_TWO | http://domain.com/1,http://domain.com/2,http://domain.com/3,http://domain.com/xyz |
// |---------------|-----------------------------------------------------------------------------------|
// When I send a purge request to /souin-api/souin with the headers
// Surrogate-Keys: GROUP_KEY_ONE
// Then the cache will be purged for the list
// * http://domain.com/
// * http://domain.com/1
// * http://domain.com/2
// * http://domain.com/4
// And the data in the Surrogate-Keys table storage will contain
// |---------------|-------------------------------------------|
// | YKey          | URLs                                      |
// |---------------|-------------------------------------------|
// | GROUP_KEY_ONE |                                           |
// | GROUP_KEY_TWO | http://domain.com/3,http://domain.com/xyz |
// |---------------|-------------------------------------------|
//
// Another example
// Given the Surrogate-Keys data as
// |---------------|-----------------------------------------------------------------------------------|
// | YKey          | URLs                                                                              |
// |---------------|-----------------------------------------------------------------------------------|
// | GROUP_KEY_ONE | http://domain.com/,http://domain.com/1,http://domain.com/2,http://domain.com/4    |
// | GROUP_KEY_TWO | http://domain.com/1,http://domain.com/2,http://domain.com/3,http://domain.com/xyz |
// |---------------|-----------------------------------------------------------------------------------|
// When I send a purge request to /souin-api/souin
// Surrogate-Keys: GROUP_KEY_ONE, GROUP_KEY_TWO
// Then the cache will be purged for the list
// * http://domain.com/
// * http://domain.com/1
// * http://domain.com/2
// * http://domain.com/4
// * http://domain.com/xyz
// And the data in the Surrogate-Keys table storage will contain
// |---------------|------|
// | YKey          | URLs |
// |---------------|------|
// | GROUP_KEY_ONE |      |
// | GROUP_KEY_TWO |      |
// |---------------|------|
//
// If the Surrogate Storage is configured with the dynamic boolean value, then it will handle and store all Surrogate-Keys
// sent by the server on a specific resource.
// Given the Surrogate-Keys data as
// |---------------|------|
// | YKey          | URLs |
// |---------------|------|
// | GROUP_KEY_ONE |      |
// | GROUP_KEY_TWO |      |
// |---------------|------|
// When you send a request to /service_1/my_first_resource
// Then the server response contains the following headers
// Surrogate-Keys: GROUP_KEY_NEW, another_one
// Then the data in the Surrogate-Keys table storage will contain
// |---------------|------------------------------|
// | YKey          | URLs                         |
// |---------------|------------------------------|
// | another_one   | /service_1/my_first_resource |
// | GROUP_KEY_NEW | /service_1/my_first_resource |
// | GROUP_KEY_ONE |                              |
// | GROUP_KEY_TWO |                              |
// |---------------|------------------------------|

import (
	"regexp"
	"strings"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/dgraph-io/ristretto"
)

func ParseHeaders(value string) []string {
	r, _ := regexp.Compile(",( +)?")
	return strings.Fields(r.ReplaceAllString(value, " "))
}

// InitializeSurrogate will initialize the Surrogate-Keys storage system
func InitializeSurrogate(keys map[string]configurationtypes.YKey, configurationInterface configurationtypes.AbstractConfigurationInterface) *SurrogateStorage {
	if len(keys) == 0 {
		return nil
	}

	storage, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     1 << 30,
		BufferItems: 64,
	})

	for key := range keys {
		storage.Set(key, "", 1)
	}

	return &SurrogateStorage{Cache: storage, Keys: keys}
}
