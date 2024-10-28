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
	template       string
	overrides      []map[*regexp.Regexp]keyContext

	initializer func(r *http.Request) *http.Request
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
	g.template = k.Template
	g.headers = k.Headers

	g.overrides = make([]map[*regexp.Regexp]keyContext, 0)

	// for _, cacheKey := range c.GetCacheKeys() {
	// 	for r, v := range cacheKey {
	// 		g.overrides = append(g.overrides, map[*regexp.Regexp]keyContext{r.Regexp: {
	// 			disable_body:   v.DisableBody,
	// 			disable_host:   v.DisableHost,
	// 			disable_method: v.DisableMethod,
	// 			disable_query:  v.DisableQuery,
	// 			disable_scheme: v.DisableScheme,
	// 			hash:           v.Hash,
	// 			displayable:    !v.Hide,
	// 			template:       v.Template,
	// 			headers:        v.Headers,
	// 		}})
	// 	}
	// }

	g.initializer = func(r *http.Request) *http.Request {
		return r
	}
}

func parseKeyInformations(req *http.Request, kCtx keyContext) (query, body, host, scheme, method, headerValues string, headers []string, displayable, hash bool) {
	displayable = kCtx.displayable
	hash = kCtx.hash

	if !kCtx.disable_query && len(req.URL.RawQuery) > 0 {
		query += "?" + req.URL.RawQuery
	}

	if !kCtx.disable_body {
		body = req.Context().Value(HashBody).(string)
	}

	if !kCtx.disable_host {
		host = req.Host + "-"
	}

	if !kCtx.disable_scheme {
		scheme = "http-"
		if req.TLS != nil {
			scheme = "https-"
		}
	}

	if !kCtx.disable_method {
		method = req.Method + "-"
	}

	headers = kCtx.headers
	for _, hn := range kCtx.headers {
		headerValues += "-" + req.Header.Get(hn)
	}

	return
}

func (g *keyContext) computeKey(req *http.Request) (key string, headers []string, hash, displayable bool) {
	key = req.URL.Path
	query, body, host, scheme, method, headerValues, headers, displayable, hash := parseKeyInformations(req, *g)

	hasOverride := false
	for _, current := range g.overrides {
		for k, v := range current {
			if k.MatchString(req.RequestURI) {
				query, body, host, scheme, method, headerValues, headers, displayable, hash = parseKeyInformations(req, v)
				hasOverride = true
				break
			}
		}

		if hasOverride {
			break
		}
	}

	key = method + scheme + host + key + query + body + headerValues

	return
}

func (g *keyContext) SetContext(req *http.Request) *http.Request {
	rq := g.initializer(req)
	key, headers, hash, displayable := g.computeKey(rq)

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
