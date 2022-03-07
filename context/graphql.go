package context

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

const (
	GraphQL           ctxKey = "GRAPHQL"
	HashBody          ctxKey = "HASH_BODY"
	IsMutationRequest ctxKey = "IS_MUTATION_REQUEST"
)

type graphQLContext struct {
	custom bool
}

func (g *graphQLContext) SetupContext(c configurationtypes.AbstractConfigurationInterface) {
	if len(c.GetDefaultCache().GetAllowedHTTPVerbs()) != 0 {
		g.custom = true
	}
	c.GetLogger().Debug("Enable GraphQL logic due to your custom HTTP verbs setup.")
}

func isMutation(b []byte) bool {
	return len(b) > 18 && string(b[:18]) == `{"query":"mutation`
}

func (g *graphQLContext) SetContext(req *http.Request) *http.Request {
	rq := req.WithContext(context.WithValue(req.Context(), GraphQL, g.custom))
	rq = rq.WithContext(context.WithValue(rq.Context(), HashBody, ""))
	rq = rq.WithContext(context.WithValue(rq.Context(), IsMutationRequest, false))

	if g.custom && req.Body != nil {
		b, _ := ioutil.ReadAll(req.Body)
		if isMutation(b) {
			rq = rq.WithContext(context.WithValue(rq.Context(), IsMutationRequest, true))
		} else {
			h := sha256.New()
			h.Write(b)
			rq = rq.WithContext(context.WithValue(rq.Context(), HashBody, fmt.Sprintf("-%x", h.Sum(nil))))
		}
	}

	return rq
}

var _ ctx = (*graphQLContext)(nil)
