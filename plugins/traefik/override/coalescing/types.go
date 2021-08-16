package coalescing

import (
	"net/http"
)

// RequestCoalescing handle the coalescing system
type RequestCoalescing struct {}

// RequestCoalescingInterface is the interface
type RequestCoalescingInterface interface {
	Temporise(*http.Request, http.ResponseWriter, func(http.ResponseWriter, *http.Request) error)
}
