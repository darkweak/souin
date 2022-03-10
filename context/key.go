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

type keyContext struct{}

func (g *keyContext) SetupContext(_ configurationtypes.AbstractConfigurationInterface) {}

func (g *keyContext) SetContext(req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), Key, fmt.Sprintf("%s-%s-%s%s", req.Method, req.Host, req.RequestURI, req.Context().Value(HashBody).(string))))
}

var _ ctx = (*keyContext)(nil)
