package context

import (
	"context"
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

const SupportedMethod ctxKey = "souin_ctx.SUPPORTED_METHOD"

var defaultVerbs []string = []string{http.MethodGet, http.MethodHead}

type methodContext struct {
	allowedVerbs []string
	custom       bool
}

func (*methodContext) SetContextWithBaseRequest(req *http.Request, _ *http.Request) *http.Request {
	return req
}

func (m *methodContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	m.allowedVerbs = defaultVerbs
	if len(c.GetDefaultCache().GetAllowedHTTPVerbs()) != 0 {
		m.allowedVerbs = c.GetDefaultCache().GetAllowedHTTPVerbs()
		m.custom = true
	}
	c.GetLogger().Debugf("Allow %d method(s). %v.", len(m.allowedVerbs), m.allowedVerbs)
}

func (m *methodContext) SetContext(req *http.Request) *http.Request {
	v := false

	for _, a := range m.allowedVerbs {
		if req.Method == a {
			v = true
			break
		}
	}

	return req.WithContext(context.WithValue(req.Context(), SupportedMethod, v))
}

var _ ctx = (*methodContext)(nil)
