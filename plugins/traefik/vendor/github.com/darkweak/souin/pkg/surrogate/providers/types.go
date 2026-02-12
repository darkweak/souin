package providers

import (
	"net/http"
)

// SurrogateInterface represents the interface to implement to be part
type SurrogateInterface interface {
	getHeaderSeparator() string
	getOrderedSurrogateKeyHeadersCandidate() []string
	getOrderedSurrogateControlHeadersCandidate() []string
	GetSurrogateControl(http.Header) (string, string)
	GetSurrogateControlName() string
	getSurrogateKey(http.Header) string
	Purge(http.Header) (cacheKeys []string, surrogateKeys []string)
	Invalidate(method string, h http.Header)
	purgeTag(string) []string
	Store(*http.Response, string, string) error
	storeTag(string, string)
	ParseHeaders(string) []string
	List() map[string]string
	candidateStore(string) bool
	Destruct() error
}
