package context

import (
	"context"
	"net/http"
	"regexp"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	Key            ctxKey = "CACHE_KEY"
	DisplayableKey ctxKey = "DISPLAYABLE_KEY"
	IgnoredHeaders ctxKey = "IGNORE_HEADERS"
)

type keyContext struct {
	disable_body   bool
	disable_host   bool
	disable_method bool
	displayable    bool
	headers        []string
	overrides      map[*regexp.Regexp]keyContext
}

func (g *keyContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	k := c.GetDefaultCache().GetKey()
	g.disable_body = k.DisableBody
	g.disable_host = k.DisableHost
	g.disable_method = k.DisableMethod
	g.displayable = !k.Hide
	g.headers = k.Headers

	g.overrides = make(map[*regexp.Regexp]keyContext)

	for r, v := range c.GetCacheKeys() {
		g.overrides[r.Regexp] = keyContext{
			disable_body:   v.DisableBody,
			disable_host:   v.DisableHost,
			disable_method: v.DisableMethod,
			displayable:    v.Hide,
			headers:        v.Headers,
		}
	}
}

func (g *keyContext) SetContext(req *http.Request) *http.Request {
	key := req.RequestURI
	var headers []string

	scheme := req.URL.Scheme + "-"
	body := ""
	host := ""
	method := ""
	headerValues := ""
	displayable := g.displayable

	if !g.disable_body {
		body = req.Context().Value(HashBody).(string)
	}

	if !g.disable_host {
		host = req.Host + "-"
	}

	if !g.disable_method {
		method = req.Method + "-"
	}

	headers = g.headers
	for _, hn := range g.headers {
		headerValues += "-" + req.Header.Get(hn)
	}

	for k, v := range g.overrides {
		if k.MatchString(req.RequestURI) {
			displayable = v.displayable
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
			if len(v.headers) > 0 {
				headerValues = ""
				for _, hn := range v.headers {
					headers = v.headers
					headerValues += "-" + req.Header.Get(hn)
				}
			}
			break
		}
	}

	return req.WithContext(
		context.WithValue(
			context.WithValue(
				context.WithValue(
					req.Context(),
					Key,
					method+scheme+host+key+body+headerValues,
				),
				IgnoredHeaders,
				headers,
			),
			DisplayableKey,
			displayable,
		),
	)
}

var _ ctx = (*keyContext)(nil)
