package providers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/darkweak/souin/configurationtypes"
)

// VarySeparator will separate vary headers from the plain URL
const VarySeparator = "{-VARY-}"
const DecodedHeaderSeparator = ";"
const encodedHeaderSeparator = "%3B"
const StalePrefix = "STALE_"

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configurationtypes.AbstractConfigurationInterface) *Cache {
	r, _ := CacheConnectionFactory(configuration)
	e := r.Init()
	if e != nil {
		panic(e)
	}
	return r
}

func varyVoter(baseKey string, req *http.Request, currentKey string) bool {
	if currentKey == baseKey {
		return true
	}

	if strings.Contains(currentKey, VarySeparator) && strings.HasPrefix(currentKey, baseKey+VarySeparator) {
		list := currentKey[(strings.LastIndex(currentKey, VarySeparator) + len(VarySeparator)):]
		if len(list) == 0 {
			return false
		}

		for _, item := range strings.Split(list, ";") {
			index := strings.LastIndex(item, ":")
			if len(item) < index+1 {
				return false
			}

			hVal := item[index+1:]
			if strings.Contains(hVal, encodedHeaderSeparator) {
				hVal, _ = url.QueryUnescape(hVal)
			}
			if req.Header.Get(item[:index]) != hVal {
				return false
			}
		}

		return true
	}

	return false
}
