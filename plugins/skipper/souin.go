package souin

import (
	"encoding/json"
	"net/http"

	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/souin/plugins"
	"github.com/zalando/skipper/filters"
)

type httpcacheMiddleware struct {
	*middleware.SouinBaseHandler
}

func NewHTTPCacheFilter() filters.Spec {
	return &httpcacheMiddleware{}
}

func (s *httpcacheMiddleware) Name() string { return "httpcache" }

func (s *httpcacheMiddleware) CreateFilter(config []interface{}) (filters.Filter, error) {
	if len(config) < 1 || config[0] == nil || config[0] == "" {
		return nil, filters.ErrInvalidFilterParameters
	}
	configuration, ok := config[0].(string)
	if !ok {
		return nil, filters.ErrInvalidFilterParameters
	}
	var c plugins.BaseConfiguration
	if e := json.Unmarshal([]byte(configuration), &c); e != nil {
		return nil, filters.ErrInvalidFilterParameters
	}

	return &httpcacheMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&c),
	}, nil
}

func (s *httpcacheMiddleware) Request(ctx filters.FilterContext) {
	rw := ctx.ResponseWriter()
	rq := ctx.Request()

	s.ServeHTTP(rw, rq, func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
}

func (s *httpcacheMiddleware) Response(ctx filters.FilterContext) {}

func NewSouinFilter() filters.Spec {
	return &httpcacheMiddleware{}
}
