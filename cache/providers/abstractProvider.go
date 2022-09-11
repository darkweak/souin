package providers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/configurationtypes"
)

// VarySeparator will separate vary headers from the plain URL
const VarySeparator = "{-VARY-}"
const stalePrefix = "STALE_"

// InitializeProvider allow to generate the providers array according to the configuration
func InitializeProvider(configuration configurationtypes.AbstractConfigurationInterface) types.AbstractProviderInterface {
	var r types.AbstractProviderInterface
	if configuration.GetDefaultCache().GetDistributed() {
		fmt.Printf("%+v\n\n", configuration.GetDefaultCache().GetRedis())
		if configuration.GetDefaultCache().GetEtcd().Configuration != nil {
			r, _ = EtcdConnectionFactory(configuration)
		} else if configuration.GetDefaultCache().GetRedis().Configuration != nil || configuration.GetDefaultCache().GetRedis().URL != "" {
			r, _ = RedisConnectionFactory(configuration)
		} else {
			if configuration.GetDefaultCache().GetOlric().URL != "" {
				r, _ = OlricConnectionFactory(configuration)
			} else {
				r, _ = EmbeddedOlricConnectionFactory(configuration)
			}
		}
	} else if configuration.GetDefaultCache().GetNuts().Configuration != nil || configuration.GetDefaultCache().GetNuts().Path != "" {
		r, _ = NutsConnectionFactory(configuration)
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

	if strings.Contains(currentKey, VarySeparator) && strings.HasPrefix(currentKey, baseKey+VarySeparator) {
		list := currentKey[(strings.LastIndex(currentKey, VarySeparator) + len(VarySeparator)):]
		if len(list) == 0 {
			return false
		}

		for _, item := range strings.Split(list, ";") {
			index := strings.LastIndex(item, ":")
			if !(len(item) >= index+1 && req.Header.Get(item[:index]) == item[index+1:]) {
				return false
			}
		}

		return true
	}

	return false
}
