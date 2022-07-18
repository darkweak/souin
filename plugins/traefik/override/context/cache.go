package context

import (
	"context"
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

const CacheName ctxKey = "CACHE_NAME"

var defaultCacheName string = "Souin"

type cacheContext struct {
	cacheName string
}

func (cc *cacheContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	cc.cacheName = defaultCacheName
	if c.GetDefaultCache().GetCacheName() != "" {
		cc.cacheName = c.GetDefaultCache().GetCacheName()
	}
}

func (cc *cacheContext) SetContext(req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), CacheName, cc.cacheName))
}

var _ ctx = (*cacheContext)(nil)
