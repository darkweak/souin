package coalescing

import (
	"net/http"

	"github.com/go-chi/stampede"
)

// RequestCoalescing handle the coalescing system
type RequestCoalescing struct {
	*stampede.Cache
}

// RequestCoalescingInterface is the interface
type RequestCoalescingInterface interface {
	Temporize(*http.Request, http.ResponseWriter, func(http.ResponseWriter, *http.Request) error)
}
