package context

import (
	"context"
	"net/http"
	"time"

	"github.com/darkweak/souin/configurationtypes"
)

const Now ctxKey = "souin_ctx.NOW"

type nowContext struct{}

func (*nowContext) SetContextWithBaseRequest(req *http.Request, _ *http.Request) *http.Request {
	return req
}

func (cc *nowContext) SetupContext(_ configurationtypes.AbstractConfigurationInterface) {}

func (cc *nowContext) SetContext(req *http.Request) *http.Request {
	var now time.Time
	var e error

	now, e = time.Parse(time.RFC1123, req.Header.Get("Date"))

	if e != nil {
		now := time.Now()
		req.Header.Set("Date", now.Format(time.RFC1123))
	}

	return req.WithContext(context.WithValue(req.Context(), Now, now))
}

var _ ctx = (*nowContext)(nil)
