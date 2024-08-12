package context

import (
	"context"
	"net/http"
	"regexp"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	Key            ctxKey = "souin_ctx.CACHE_KEY"
	DisplayableKey ctxKey = "souin_ctx.DISPLAYABLE_KEY"
	IgnoredHeaders ctxKey = "souin_ctx.IGNORE_HEADERS"
)

type keyContext struct {
	disable_body   bool
	disable_host   bool
	disable_method bool
	disable_query  bool
	disable_scheme bool
	hash           bool
	displayable    bool
	headers        []string
	overrides      []map[*regexp.Regexp]keyContext
}

func (g *keyContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	k := c.GetDefaultCache().GetKey()
	g.disable_body = k.DisableBody
	g.disable_host = k.DisableHost
	g.disable_method = k.DisableMethod
	g.disable_query = k.DisableQuery
	g.disable_scheme = k.DisableScheme
	g.hash = k.Hash
	g.displayable = !k.Hide
	g.headers = k.Headers

	g.overrides = make([]map[*regexp.Regexp]keyContext, 0)

	for _, cacheKey := range c.GetCacheKeys() {
		for r, v := range cacheKey {
			g.overrides = append(g.overrides, map[*regexp.Regexp]keyContext{r.Regexp: {
				disable_body:   v.DisableBody,
				disable_host:   v.DisableHost,
				disable_method: v.DisableMethod,
				disable_query:  v.DisableQuery,
				disable_scheme: v.DisableScheme,
				hash:           v.Hash,
				displayable:    !v.Hide,
				headers:        v.Headers,
			}})
		}
	}
}

func (g *keyContext) SetContext(req *http.Request) *http.Request {
	key := req.URL.Path
	var headers []string

	scheme := "http-"
	if req.TLS != nil {
		scheme = "https-"
	}
	query := ""
	body := ""
	host := ""
	method := ""
	headerValues := ""
	displayable := g.displayable

	if !g.disable_query && len(req.URL.RawQuery) > 0 {
		query += "?" + req.URL.RawQuery
	}

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

	hasOverride := false
	for _, current := range g.overrides {
		for k, v := range current {
			if k.MatchString(req.RequestURI) {
				displayable = v.displayable
				host = ""
				method = ""
				query = ""
				if !v.disable_query && len(req.URL.RawQuery) > 0 {
					query = "?" + req.URL.RawQuery
				}
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
				hasOverride = true
				break
			}
		}

		if hasOverride {
			break
		}
	}

	return req.WithContext(
		context.WithValue(
			context.WithValue(
				context.WithValue(
					req.Context(),
					Key,
					method+scheme+host+key+query+body+headerValues,
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
