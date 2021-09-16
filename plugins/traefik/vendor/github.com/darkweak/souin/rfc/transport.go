package rfc

import (
	"github.com/darkweak/souin/cache/surrogate/providers"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"net/http"
)

// VaryTransport type
type VaryTransport types.Transport

// IsVaryCacheable determines if it's cacheable
func IsVaryCacheable(req *http.Request) bool {
	method := req.Method
	rangeHeader := req.Header.Get("range")
	return (method == http.MethodGet || method == http.MethodHead) && rangeHeader == ""
}

// NewTransport returns a new Transport with the
// provided Cache implementation and MarkCachedResponses set to true
func NewTransport(p types.AbstractProviderInterface, ykeyStorage *ykeys.YKeyStorage, surrogateStorage providers.SurrogateInterface) *VaryTransport {
	return &VaryTransport{
		Provider:               p,
		CoalescingLayerStorage: types.InitializeCoalescingLayerStorage(),
		MarkCachedResponses:    true,
		YkeyStorage:            ykeyStorage,
		SurrogateStorage:       surrogateStorage,
	}
}
