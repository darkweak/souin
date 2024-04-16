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
	Hashed         ctxKey = "souin_ctx.HASHED"
)

type keyContext struct {
	disable_body   bool
	disable_host   bool
	disable_method bool
	disable_query  bool
	disable_scheme bool
	displayable    bool
	hash           bool
	headers        []string
	overrides      []map[*regexp.Regexp]keyContext
}

func (*keyContext) SetContextWithBaseRequest(req *http.Request, _ *http.Request) *http.Request {
	return req
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

	hash := g.hash
	query := ""
	scheme := ""
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

	if !g.disable_scheme {
		scheme = "http-"
		if req.TLS != nil {
			scheme = "https-"
		}
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
				scheme = ""
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
				if !v.disable_scheme {
					scheme = "http-"
					if req.TLS != nil {
						scheme = "https-"
					}
				}
				if len(v.headers) > 0 {
					headerValues = ""
					for _, hn := range v.headers {
						headers = v.headers
						headerValues += "-" + req.Header.Get(hn)
					}
				}
				if v.hash {
					hash = true
				}
				hasOverride = true
				break
			}
		}

		if hasOverride {
			break
		}
	}

	key = method + scheme + host + key + query + body + headerValues

	return req.WithContext(
		context.WithValue(
			context.WithValue(
				context.WithValue(
					context.WithValue(
						req.Context(),
						Key,
						key,
					),
					Hashed,
					hash,
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
