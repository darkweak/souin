package souin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"github.com/zalando/skipper/filters"
)

type (
	httpcache struct {
		plugins.SouinBasePlugin
		Configuration *Configuration
		bufPool       *sync.Pool
	}
)

func NewSouinFilter() filters.Spec {
	return &httpcache{}
}

func (s *httpcache) Name() string { return "httpcache" }

func (s *httpcache) CreateFilter(config []interface{}) (filters.Filter, error) {
	if len(config) < 1 || config[0] == nil || config[0] == "" {
		return nil, filters.ErrInvalidFilterParameters
	}
	configuration, ok := config[0].(string)
	if !ok {
		return nil, filters.ErrInvalidFilterParameters
	}
	var c Configuration
	if e := json.Unmarshal([]byte(configuration), &c); e != nil {
		return nil, filters.ErrInvalidFilterParameters
	}

	s.Configuration = &c
	s.bufPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	s.Retriever = plugins.DefaultSouinPluginInitializerFromConfiguration(&c)
	s.RequestCoalescing = coalescing.Initialize()
	s.MapHandler = api.GenerateHandlerMap(s.Configuration, s.Retriever.GetTransport())
	return s, nil
}

func (s *httpcache) Request(ctx filters.FilterContext) {
	rw := ctx.ResponseWriter()
	req := s.Retriever.GetContext().Method.SetContext(ctx.Request())
	if !plugins.CanHandle(req, s.Retriever) {
		rw.Header().Add("Cache-Status", "Souin; fwd=uri-miss")
		return
	}

	if b, handler := s.HandleInternally(req); b {
		handler(rw, req)
		return
	}

	writer := overrideWriter{
		&plugins.CustomWriter{
			Response: ctx.Response(),
			Buf:      s.bufPool.Get().(*bytes.Buffer),
			Rw:       rw,
		},
	}
	req.Header.Set("Date", time.Now().UTC().Format(time.RFC1123))
	req = s.Retriever.GetContext().SetContext(req)
	if plugins.HasMutation(req, rw) {
		return
	}

	plugins.DefaultSouinPluginCallback(writer, req, s.Retriever, nil, func(_ http.ResponseWriter, _ *http.Request) error {
		return nil
	})

	if writer.Response != nil {
		ctx.Serve(writer.Response)
	}
}

func (s *httpcache) Response(ctx filters.FilterContext) {
	req := ctx.Request()
	rw := ctx.ResponseWriter()
	res := ctx.Response()
	customWriter := &plugins.CustomWriter{
		Response: ctx.Response(),
		Buf:      s.bufPool.Get().(*bytes.Buffer),
		Rw:       rw,
	}
	req.Response = res
	req = s.Retriever.GetContext().Method.SetContext(req)
	if !plugins.CanHandle(req, s.Retriever) {
		res.Header.Add("Cache-Status", "Souin; fwd=uri-miss")
		return
	}

	var e error
	req = s.Retriever.GetContext().SetContext(req)
	if plugins.HasMutation(req, rw) {
		return
	}
	if req.Response, e = s.Retriever.GetTransport().(*rfc.VaryTransport).UpdateCacheEventually(req); e != nil {
		return
	}

	for h, v := range customWriter.Response.Header {
		if len(v) > 0 {
			customWriter.Response.Header.Set(h, strings.Join(v, ", "))
		}
	}
	ctx.Serve(customWriter.Response)
}
