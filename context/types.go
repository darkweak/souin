package context

import (
	"net/http"

	"github.com/darkweak/souin/configurationtypes"
)

type (
	ctxKey string

	ctx interface {
		SetupContext(c configurationtypes.AbstractConfigurationInterface)
		SetContext(req *http.Request) *http.Request
	}

	Context struct {
		GraphQL ctx
		Key     ctx
		Method  ctx
	}
)

func GetContext() *Context {
	return &Context{
		GraphQL: &graphQLContext{},
		Key:     &keyContext{},
		Method:  &methodContext{},
	}
}

func (c *Context) Init(co configurationtypes.AbstractConfigurationInterface) {
	c.GraphQL.SetupContext(co)
	c.Key.SetupContext(co)
	c.Method.SetupContext(co)
}

func (c *Context) SetContext(req *http.Request) *http.Request {
	req = c.GraphQL.SetContext(req)
	return c.Key.SetContext(req)
}
