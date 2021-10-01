package providers

import (
	"net/http"
	"regexp"
)

// SurrogateInterface represents the interface to implement to be part
type SurrogateInterface interface {
	getHeaderSeparator() string
	getOrderedSurrogateKeyHeadersCandidate() []string
	getOrderedSurrogateControlHeadersCandidate() []string
	getSurrogateControl(http.Header) string
	getSurrogateKey(http.Header) string
	Purge(http.Header) (cacheKeys []string, surrogateKeys []string)
	purgeTag(string) []string
	Store(*http.Request, string) error
	storeTag(string, string, *regexp.Regexp)
	ParseHeaders(string) []string
	candidateStore(string) bool
}
