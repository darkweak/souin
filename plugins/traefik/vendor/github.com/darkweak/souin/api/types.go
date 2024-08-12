package api

import (
	"net/http"
)

// EndpointInterface is the contract to be able to enable your custom endpoints
type EndpointInterface interface {
	GetBasePath() string
	IsEnabled() bool
	HandleRequest(http.ResponseWriter, *http.Request)
}
