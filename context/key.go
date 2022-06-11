package context

import (
	"context"
	"fmt"
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	Key ctxKey = "CACHE_KEY"
)

type keyContext struct {
	disable_host   bool
	disable_method bool
}

func (g *keyContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	k := c.GetDefaultCache().GetKey()
	fmt.Printf("%+v\n", k)
	g.disable_host = k.DisableHost
	g.disable_method = k.DisableMethod
}

func (g *keyContext) SetContext(req *http.Request) *http.Request {
	key := fmt.Sprintf("%s%s", req.RequestURI, req.Context().Value(HashBody).(string))
	if !g.disable_host {
		key = fmt.Sprintf("%s-%s", req.Host, key)
	}
	if !g.disable_method {
		key = fmt.Sprintf("%s-%s", req.Method, key)
	}
	return req.WithContext(context.WithValue(req.Context(), Key, key))
}

var _ ctx = (*keyContext)(nil)
