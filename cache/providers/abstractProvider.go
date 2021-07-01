package providers

import (
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
	"net/http"
	"strings"
)

const VarySeparator = "{-VARY-}"

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configurationtypes.AbstractConfigurationInterface) types.AbstractProviderInterface {
	var r types.AbstractProviderInterface
	if configuration.GetDefaultCache().GetDistributed() {
		if configuration.GetDefaultCache().GetOlric().URL != "" {
			r, _ = OlricConnectionFactory(configuration)
		} else {
			r, _ = EmbeddedOlricConnectionFactory(configuration)
		}
	} else {
		r, _ = BadgerConnectionFactory(configuration)
	}
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
		list := currentKey[(strings.LastIndex(currentKey, VarySeparator) + 1):]
		if len(list) == 0 {
			return false
		}

		for _, item := range strings.Split(list, ";") {
			index := strings.LastIndex(currentKey, ":")
			if req.Header.Get(item[:index]) != item[index+1:] {
				return false
			}
		}

		return true
	}

	return false
}
