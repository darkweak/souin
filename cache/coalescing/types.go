package coalescing

import (
	"github.com/go-chi/stampede"
	"net/http"
)

// RequestCoalescing handle the coalescing system
type RequestCoalescing struct {
	*stampede.Cache
}

// RequestCoalescingInterface is the interface
type RequestCoalescingInterface interface {
	Temporise(*http.Request, http.ResponseWriter, func(http.ResponseWriter, *http.Request) error)
}
