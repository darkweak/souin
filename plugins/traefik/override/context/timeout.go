package context

import (
	"context"
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	TimeoutCache  ctxKey = "souin_ctx.TIMEOUT_CACHE"
	TimeoutCancel ctxKey = "souin_ctx.TIMEOUT_CANCEL"
)

const (
	defaultTimeoutBackend = 10 * time.Second
	defaultTimeoutCache   = 10 * time.Millisecond
)

type timeoutContext struct {
	timeoutCache, timeoutBackend time.Duration
}

func (*timeoutContext) SetContextWithBaseRequest(req *http.Request, _ *http.Request) *http.Request {
	return req
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
}

func (t *timeoutContext) SetContext(req *http.Request) *http.Request {
	ctx, cancel := context.WithTimeout(req.Context(), t.timeoutBackend)
	return req.WithContext(context.WithValue(context.WithValue(ctx, TimeoutCancel, cancel), TimeoutCache, t.timeoutCache))
}

var _ ctx = (*timeoutContext)(nil)
