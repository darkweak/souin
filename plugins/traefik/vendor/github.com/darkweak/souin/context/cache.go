package context

import (
	"context"
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/pquerna/cachecontrol/cacheobject"
)

const (
	CacheName           ctxKey = "souin_ctx.CACHE_NAME"
	RequestCacheControl ctxKey = "souin_ctx.REQUEST_CACHE_CONTROL"
)

var defaultCacheName string = "Souin"

type cacheContext struct {
	cacheName string
}

func (*cacheContext) SetContextWithBaseRequest(req *http.Request, _ *http.Request) *http.Request {
	return req
}

func (cc *cacheContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	cc.cacheName = defaultCacheName
	if c.GetDefaultCache().GetCacheName() != "" {
		cc.cacheName = c.GetDefaultCache().GetCacheName()
	}
	c.GetLogger().Sugar().Debugf("Set %s as Cache-Status name", cc.cacheName)
}

func (cc *cacheContext) SetContext(req *http.Request) *http.Request {
	co, _ := cacheobject.ParseRequestCacheControl(req.Header.Get("Cache-Control"))
	return req.WithContext(context.WithValue(context.WithValue(req.Context(), CacheName, cc.cacheName), RequestCacheControl, co))
}

var _ ctx = (*cacheContext)(nil)
