package types

import (
	"net/http"
	"regexp"

	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/context"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/souin/pkg/surrogate/providers"
)

// TransportInterface interface
type TransportInterface interface {
	GetProvider() types.Storer
	RoundTrip(req *http.Request) (resp *http.Response, err error)
	SetURL(url configurationtypes.URL)
	UpdateCacheEventually(req *http.Request) (resp *http.Response, err error)
	GetCoalescingLayerStorage() *CoalescingLayerStorage
	GetYkeyStorage() *ykeys.YKeyStorage
	GetSurrogateKeys() providers.SurrogateInterface
	SetSurrogateKeys(providers.SurrogateInterface)
}

// Transport is an implementation of http.RoundTripper that will return values from a cache
// where possible (avoiding a network request) and will additionally add validators (etag/if-modified-since)
// to repeated requests allowing servers to return 304 / Not Modified
type Transport struct {
	// The RoundTripper interface actually used to make requests
	// If nil, http.DefaultTransport is used
	Transport              http.RoundTripper
	Provider               types.Storer
	ConfigurationURL       configurationtypes.URL
	MarkCachedResponses    bool
	CoalescingLayerStorage *CoalescingLayerStorage
	YkeyStorage            *ykeys.YKeyStorage
	SurrogateStorage       providers.SurrogateInterface
}

// RetrieverResponsePropertiesInterface interface
type RetrieverResponsePropertiesInterface interface {
	GetProvider() types.Storer
	GetConfiguration() configurationtypes.AbstractConfigurationInterface
	GetMatchedURL() configurationtypes.URL
	SetMatchedURL(url configurationtypes.URL)
	SetMatchedURLFromRequest(*http.Request)
	GetRegexpUrls() *regexp.Regexp
	GetTransport() TransportInterface
	SetTransport(TransportInterface)
	GetExcludeRegexp() *regexp.Regexp
	GetContext() *context.Context
}

// RetrieverResponseProperties struct
type RetrieverResponseProperties struct {
	Provider      types.Storer
	Configuration configurationtypes.AbstractConfigurationInterface
	MatchedURL    configurationtypes.URL
	RegexpUrls    regexp.Regexp
	Transport     TransportInterface
	ExcludeRegex  *regexp.Regexp
	Context       *context.Context
}

// GetProvider interface
func (r *RetrieverResponseProperties) GetProvider() types.Storer {
	return r.Provider
}

// GetConfiguration get the configuration
func (r *RetrieverResponseProperties) GetConfiguration() configurationtypes.AbstractConfigurationInterface {
	return r.Configuration
}

// GetMatchedURL get the matched url
func (r *RetrieverResponseProperties) GetMatchedURL() configurationtypes.URL {
	return r.MatchedURL
}

// SetMatchedURL set the matched url
func (r *RetrieverResponseProperties) SetMatchedURL(url configurationtypes.URL) {
	r.MatchedURL = url
}

// SetMatchedURLFromRequest set the matched url from the request
func (r *RetrieverResponseProperties) SetMatchedURLFromRequest(req *http.Request) {
	path := req.Host + req.URL.Path
	regexpURL := r.GetRegexpUrls().FindString(path)
	url := configurationtypes.URL{
		TTL:                 configurationtypes.Duration{Duration: r.GetConfiguration().GetDefaultCache().GetTTL()},
		Headers:             r.GetConfiguration().GetDefaultCache().GetHeaders(),
		DefaultCacheControl: r.GetConfiguration().GetDefaultCache().GetDefaultCacheControl(),
	}
	if regexpURL != "" {
		u := r.GetConfiguration().GetUrls()[regexpURL]
		if u.TTL.Duration != 0 {
			url.TTL = u.TTL
		}
		if len(u.Headers) != 0 {
			url.Headers = u.Headers
		}
	}
	r.GetTransport().SetURL(url)
	r.SetMatchedURL(url)
}

// GetRegexpUrls get the regexp urls
func (r *RetrieverResponseProperties) GetRegexpUrls() *regexp.Regexp {
	return &r.RegexpUrls
}

// GetTransport get the transport according to the RFC
func (r *RetrieverResponseProperties) GetTransport() TransportInterface {
	return r.Transport
}

// SetTransport set the transport
func (r *RetrieverResponseProperties) SetTransport(transportInterface TransportInterface) {
	r.Transport = transportInterface
}

// GetExcludeRegexp get the excluded regexp
func (r *RetrieverResponseProperties) GetExcludeRegexp() *regexp.Regexp {
	return r.ExcludeRegex
}

// GetContext get the different contexts to init/use
func (r *RetrieverResponseProperties) GetContext() *context.Context {
	return r.Context
}
