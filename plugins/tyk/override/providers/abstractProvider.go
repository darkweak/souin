package providers

import (
	"net/http"
	"strings"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// VarySeparator will separate vary headers from the plain URL
const VarySeparator = "{-VARY-}"

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configurationtypes.AbstractConfigurationInterface) types.AbstractProviderInterface {
	var r types.AbstractProviderInterface
	r, _ = EmbeddedOlricConnectionFactory(configuration)
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

	if strings.Contains(currentKey, VarySeparator) {
		list := currentKey[(strings.LastIndex(currentKey, VarySeparator) + len(VarySeparator)):]
		if len(list) == 0 {
			return false
		}

		for _, item := range strings.Split(list, ";") {
			index := strings.LastIndex(item, ":")
			if req.Header.Get(item[:index]) != item[index+1:] {
				return false
			}
		}

		return true
	}

	return false
}
