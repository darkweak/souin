package coalescing

import (
	"github.com/darkweak/souin/cache/types"
	"golang.org/x/sync/singleflight"
	"net/http"
)

// RequestCoalescingChannelItem is the item sent to the channel
type RequestCoalescingChannelItem struct {
	Rq *http.Request
	Rw http.ResponseWriter
}

// RequestCoalescing handle channels map
type RequestCoalescing struct {
	requestGroup singleflight.Group
	store        *types.CoalescingLayerStorage
}

// RequestCoalescingInterface is the interface
type RequestCoalescingInterface interface {
	Temporise(*http.Request, http.ResponseWriter, func(http.ResponseWriter, *http.Request) error)
	GetCoalescingLayerStorage() *types.CoalescingLayerStorage
}
