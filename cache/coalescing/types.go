package coalescing

import (
	"github.com/darkweak/souin/cache/types"
	"net/http"
)

// RequestCoalescingChannelItem is the item sent to the channel
type RequestCoalescingChannelItem struct {
	Rq *http.Request
	Rw http.ResponseWriter
}

// RequestCoalescing handle channels map
type RequestCoalescing struct {
	channels map[string]chan RequestCoalescingChannelItem
}

// RequestCoalescingInterface is the interface
type RequestCoalescingInterface interface {
	Drop(string)
	Reset() *RequestCoalescing
	Resolve(types.ReverseResponse, *http.Request, http.ResponseWriter)
	Temporise(*http.Request, http.ResponseWriter, types.RetrieverResponsePropertiesInterface)
}
