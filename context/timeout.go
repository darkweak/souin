package context

import (
	"context"
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	TimeoutCache  ctxKey = "TIMEOUT_CACHE"
	TimeoutCancel ctxKey = "TIMEOUT_CANCEL"
)

const (
	defaultTimeoutBackend = 10 * time.Second
	defaultTimeoutCache   = 10 * time.Millisecond
)

type timeoutContext struct {
	timeoutCache, timeoutBackend time.Duration
}

func (t *timeoutContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	t.timeoutBackend = defaultTimeoutBackend
	t.timeoutCache = defaultTimeoutCache
	if c.GetDefaultCache().GetTimeout().Cache.Duration != 0 {
		t.timeoutCache = c.GetDefaultCache().GetTimeout().Cache.Duration
	}
	if c.GetDefaultCache().GetTimeout().Backend.Duration != 0 {
		t.timeoutBackend = c.GetDefaultCache().GetTimeout().Backend.Duration
	}
	c.GetLogger().Sugar().Infof("Set backend timeout to %v", t.timeoutBackend)
	c.GetLogger().Sugar().Infof("Set cache timeout to %v", t.timeoutBackend)
}

func (t *timeoutContext) SetContext(req *http.Request) *http.Request {
	ctx, cancel := context.WithTimeout(req.Context(), t.timeoutBackend)
	return req.WithContext(context.WithValue(context.WithValue(ctx, TimeoutCancel, cancel), TimeoutCache, t.timeoutCache))
}

var _ ctx = (*cacheContext)(nil)
