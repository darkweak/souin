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
	disable_body   bool
	disable_host   bool
	disable_method bool
	overrides      map[*regexp.Regexp]keyContext
}

func (g *keyContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	k := c.GetDefaultCache().GetKey()
	g.disable_body = k.DisableBody
	g.disable_host = k.DisableHost
	g.disable_method = k.DisableMethod

	g.overrides = make(map[*regexp.Regexp]keyContext)

	for r, v := range c.GetCacheKeys() {
		g.overrides[r.Regexp] = keyContext{
			disable_body:   v.DisableBody,
			disable_host:   v.DisableHost,
			disable_method: v.DisableMethod,
		}
	}
}

func (g *keyContext) SetContext(req *http.Request) *http.Request {
	key := req.URL.RequestURI()
	fmt.Println("The key =>", key)

	body := ""
	host := ""
	method := ""

	if !g.disable_body {
		body = req.Context().Value(HashBody).(string)
	}

	if !g.disable_host {
		host = req.Host + "-"
	}

	if !g.disable_method {
		method = req.Method + "-"
	}

	for k, v := range g.overrides {
		if k.MatchString(req.RequestURI) {
			host = ""
			method = ""
			if !v.disable_body {
				body = req.Context().Value(HashBody).(string)
			}
			if !v.disable_method {
				method = req.Method + "-"
			}
			if !v.disable_host {
				host = req.Host + "-"
			}
			break
		}
	}

	return req.WithContext(context.WithValue(req.Context(), Key, method+host+key+body))
}

var _ ctx = (*keyContext)(nil)
