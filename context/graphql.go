package context

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
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
		c.GetLogger().Debug("Enable GraphQL logic due to your custom HTTP verbs setup.")
	}
}

func isMutation(b []byte) bool {
	return len(b) > 18 && string(b[:18]) == `{"query":"mutation`
}

func (g *graphQLContext) SetContext(req *http.Request) *http.Request {
	ctx := req.Context()
	ctx = context.WithValue(ctx, GraphQL, g.custom)
	ctx = context.WithValue(ctx, HashBody, "")
	ctx = context.WithValue(ctx, IsMutationRequest, false)

	if g.custom && req.Body != nil {
		b := bytes.NewBuffer([]byte{})
		_, _ = io.Copy(b, req.Body)
		if b.Len() > 0 {
			if isMutation(b.Bytes()) {
				ctx = context.WithValue(ctx, IsMutationRequest, true)
			} else {
				h := sha256.New()
				h.Write(b.Bytes())
				ctx = context.WithValue(ctx, HashBody, fmt.Sprintf("-%x", h.Sum(nil)))
			}
		}
	}

	return req.WithContext(ctx)
}

var _ ctx = (*graphQLContext)(nil)
