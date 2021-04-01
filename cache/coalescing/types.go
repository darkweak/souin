package coalescing

import (
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
}

// RequestCoalescingInterface is the interface
type RequestCoalescingInterface interface {
	Temporise(*http.Request, http.ResponseWriter, func(http.ResponseWriter, *http.Request) error)
}
