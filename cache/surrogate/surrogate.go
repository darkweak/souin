package surrogate

// The Surrogate key system is like the ykey one in Varnish and the same as Fastly.
// You can send a PURGE request on the Souin API endpoint with the Surrogate header to invalidate the filled keys.
// The Surrogate header CAN contains one or multiple keys separated by a comma as mentioned in the RFC.
// e.g.
// Given the Surrogate-Key data as
// |---------------|-----------------------------------------------------------------------------------|
// | Surrogate Key | URLs                                                                              |
// |---------------|-----------------------------------------------------------------------------------|
// | GROUP_KEY_ONE | http://domain.com/,http://domain.com/1,http://domain.com/2,http://domain.com/4    |
// | GROUP_KEY_TWO | http://domain.com/1,http://domain.com/2,http://domain.com/3,http://domain.com/xyz |
// |---------------|-----------------------------------------------------------------------------------|
// When I send a purge request to /souin-api/souin with the headers
// Surrogate-Key: GROUP_KEY_ONE
// Then the cache will be purged for the list
// * http://domain.com/
// * http://domain.com/1
// * http://domain.com/2
// * http://domain.com/4
// And the data in the Surrogate-Key table storage will contain
// |---------------|-------------------------------------------|
// | Surrogate Key | URLs                                      |
// |---------------|-------------------------------------------|
// | GROUP_KEY_ONE |                                           |
// | GROUP_KEY_TWO | http://domain.com/3,http://domain.com/xyz |
// |---------------|-------------------------------------------|
//
// Another example
// Given the Surrogate-Key data as
// |---------------|-----------------------------------------------------------------------------------|
// | Surrogate Key | URLs                                                                              |
// |---------------|-----------------------------------------------------------------------------------|
// | GROUP_KEY_ONE | http://domain.com/,http://domain.com/1,http://domain.com/2,http://domain.com/4    |
// | GROUP_KEY_TWO | http://domain.com/1,http://domain.com/2,http://domain.com/3,http://domain.com/xyz |
// |---------------|-----------------------------------------------------------------------------------|
// When I send a purge request to /souin-api/souin
// Surrogate-Key: GROUP_KEY_ONE, GROUP_KEY_TWO
// Then the cache will be purged for the list
// * http://domain.com/
// * http://domain.com/1
// * http://domain.com/2
// * http://domain.com/4
// * http://domain.com/xyz
// And the data in the Surrogate-Key table storage will contain
// |---------------|------|
// | Surrogate Key | URLs |
// |---------------|------|
// | GROUP_KEY_ONE |      |
// | GROUP_KEY_TWO |      |
// |---------------|------|
//
// If the Surrogate Storage is configured with the dynamic boolean value, then it will handle and store all Surrogate-Key
// sent by the server on a specific resource.
// Given the Surrogate-Key data as
// |---------------|------|
// | Surrogate Key | URLs |
// |---------------|------|
// | GROUP_KEY_ONE |      |
// | GROUP_KEY_TWO |      |
// |---------------|------|
// When you send a request to /service_1/my_first_resource
// Then the server response contains the following headers
// Surrogate-Key: GROUP_KEY_NEW, another_one
// Then the data in the Surrogate-Key table storage will contain
// |---------------|------------------------------|
// | Surrogate Key | URLs                         |
// |---------------|------------------------------|
// | another_one   | /service_1/my_first_resource |
// | GROUP_KEY_NEW | /service_1/my_first_resource |
// | GROUP_KEY_ONE |                              |
// | GROUP_KEY_TWO |                              |
// |---------------|------------------------------|

import (
	"github.com/darkweak/souin/cache/surrogate/providers"
	"github.com/darkweak/souin/configurationtypes"
)

// InitializeSurrogate will initialize the Surrogate-Key storage system
func InitializeSurrogate(configurationInterface configurationtypes.AbstractConfigurationInterface) providers.SurrogateInterface {
	return providers.SurrogateFactory(configurationInterface)
}
