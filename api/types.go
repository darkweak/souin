package api

import (
	"net/http"
	"regexp"
)

// EndpointInterface is the contract to be able to enable your custom endpoints
type EndpointInterface interface {
	BulkDelete(rg *regexp.Regexp)
	Delete(key string)
	GetAll() []string
	GetBasePath() string
	IsEnabled() bool
	HandleRequest(http.ResponseWriter, *http.Request)
}
