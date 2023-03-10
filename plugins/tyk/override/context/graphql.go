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
	}
}

func isMutation(b []byte) bool {
	return len(b) > 18 && string(b[:18]) == `{"query":"mutation`
}

func (g *graphQLContext) SetContext(req *http.Request) *http.Request {
	rq := req.WithContext(context.WithValue(req.Context(), GraphQL, g.custom))
	rq = rq.WithContext(context.WithValue(rq.Context(), HashBody, ""))
	rq = rq.WithContext(context.WithValue(rq.Context(), IsMutationRequest, false))

	if g.custom && req.Body != nil {
		b := bytes.NewBuffer([]byte{})
		_, _ = io.Copy(b, req.Body)
		if b.Len() > 0 {
			if isMutation(b.Bytes()) {
				rq = rq.WithContext(context.WithValue(rq.Context(), IsMutationRequest, true))
			} else {
				h := sha256.New()
				h.Write(b.Bytes())
				rq = rq.WithContext(context.WithValue(rq.Context(), HashBody, fmt.Sprintf("-%x", h.Sum(nil))))
			}
		}
	}

	return rq
}

var _ ctx = (*graphQLContext)(nil)
