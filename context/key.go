package context

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	Key ctxKey = "CACHE_KEY"
)

type keyContext struct {
	disable_host   bool
	disable_method bool
	overrides      map[*regexp.Regexp]keyContext
}

func (g *keyContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	k := c.GetDefaultCache().GetKey()
	g.disable_host = k.DisableHost
	g.disable_method = k.DisableMethod

	g.overrides = make(map[*regexp.Regexp]keyContext)

	for r, v := range c.GetCacheKeys() {
		g.overrides[r.Regexp] = keyContext{
			disable_host:   v.DisableHost,
			disable_method: v.DisableMethod,
		}
	}
}

func (g *keyContext) SetContext(req *http.Request) *http.Request {
	key := fmt.Sprintf("%s%s", req.URL.RequestURI(), req.Context().Value(HashBody).(string))

	host := ""
	method := ""

	if !g.disable_host {
		host = req.Host + "-"
	}

	if !g.disable_method {
		method = req.Method + "-"
	}

	for k, v := range g.overrides {
		host = ""
		method = ""
		if k.MatchString(req.URL.RequestURI()) {
			if !v.disable_method {
				method = req.Method + "-"
			}
			if !v.disable_host {
				host = req.Host + "-"
			}
			break
		}
	}

	return req.WithContext(context.WithValue(req.Context(), Key, fmt.Sprintf("%s%s%s", method, host, key)))
}

var _ ctx = (*keyContext)(nil)
