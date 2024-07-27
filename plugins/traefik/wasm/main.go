package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"

	"github.com/darkweak/souin/pkg/api/prometheus"
	"github.com/darkweak/souin/pkg/middleware"
	"github.com/darkweak/storages/core"
)

// SouinTraefikMiddleware declaration.
type SouinTraefikMiddleware struct {
	next             http.Handler
	name             string
	SouinBaseHandler *middleware.SouinBaseHandler
}

type logger struct {
	api.Host
}

func (l *logger) DPanic(args ...interface{}) {
	panic("unimplemented")
}
func (l *logger) DPanicf(template string, args ...interface{}) {
	panic("unimplemented")
}
func (l *logger) Debug(args ...interface{}) {
	l.Host.Log(api.LogLevelDebug, fmt.Sprint(args...))
}
func (l *logger) Debugf(template string, args ...interface{}) {
	l.Host.Log(api.LogLevelDebug, fmt.Sprintf(template, args...))
}
func (l *logger) Error(args ...interface{}) {
	l.Host.Log(api.LogLevelError, fmt.Sprint(args...))
}
func (l *logger) Errorf(template string, args ...interface{}) {
	l.Host.Log(api.LogLevelError, fmt.Sprintf(template, args...))
}
func (l *logger) Fatal(args ...interface{}) {
	l.Host.Log(api.LogLevelWarn, fmt.Sprint(args...))
}
func (l *logger) Fatalf(template string, args ...interface{}) {
	l.Host.Log(api.LogLevelWarn, fmt.Sprintf(template, args...))
}
func (l *logger) Info(args ...interface{}) {
	l.Host.Log(api.LogLevelInfo, fmt.Sprint(args...))
}
func (l *logger) Infof(template string, args ...interface{}) {
	l.Host.Log(api.LogLevelInfo, fmt.Sprintf(template, args...))
}
func (l *logger) Panic(args ...interface{}) {
	l.Host.Log(api.LogLevelError, fmt.Sprint(args...))
}
func (l *logger) Panicf(template string, args ...interface{}) {
	l.Host.Log(api.LogLevelError, fmt.Sprintf(template, args...))
}
func (l *logger) Warn(args ...interface{}) {
	l.Host.Log(api.LogLevelWarn, fmt.Sprint(args...))
}
func (l *logger) Warnf(template string, args ...interface{}) {
	l.Host.Log(api.LogLevelWarn, fmt.Sprintf(template, args...))
}

var _ core.Logger = (*logger)(nil)

func main() {
	handler.Host.Log(api.LogLevelInfo, fmt.Sprintf("load config %v", string(handler.Host.GetConfig())))
	var config middleware.BaseConfiguration
	err := json.Unmarshal(handler.Host.GetConfig(), &config)
	config.SetLogger(&logger{Host: handler.Host})
	config.API.Souin.Enable = true
	if err != nil {
		handler.Host.Log(api.LogLevelError, fmt.Sprintf("Could not load config %v", err))
		os.Exit(1)
	}

	mw, err := New(config)
	if err != nil {
		handler.Host.Log(api.LogLevelError, fmt.Sprintf("Could not load config %v", err))
		os.Exit(1)
	}
	handler.HandleRequestFn = mw.handleRequest
	handler.HandleResponseFn = mw.handleResponse
}

// New create Souin instance.
func New(config middleware.BaseConfiguration) (*SouinTraefikMiddleware, error) {
	return &SouinTraefikMiddleware{
		SouinBaseHandler: middleware.NewHTTPCacheHandler(&config),
	}, nil
}

func apiRequestToHTTPRequest(r1 api.Request) (*http.Request, error) {
	buf := bytes.NewBuffer([]byte{})

	r1.Body().WriteTo(buf)

	r2, err := http.NewRequest(r1.GetMethod(), r1.GetURI(), bytes.NewBuffer(buf.Bytes()))
	if err != nil {
		return r2, err
	}

	r2.RequestURI = r1.GetURI()

	if r2.Host == "" {
		r2.Host, _ = r1.Headers().Get("Host")
	}

	for _, name := range r1.Headers().Names() {
		r2.Header.Set(name, strings.Join(r1.Headers().GetAll(name), ", "))
	}

	return r2, nil
}

func (s *SouinTraefikMiddleware) handleRequest(req api.Request, resp api.Response) (next bool, reqCtx uint32) {
	start := time.Now()
	defer func(s time.Time) {
		prometheus.Add(prometheus.AvgResponseTime, float64(time.Since(s).Milliseconds()))
	}(start)

	rq, _ := apiRequestToHTTPRequest(req)
	rw := newWriter(resp)
	s.SouinBaseHandler.Configuration.GetLogger().Debugf("Incomming HTTP request %#v", *rq)
	defer rw.syncHeaders()

	if b, handler := s.SouinBaseHandler.HandleInternally(rq); b {
		handler(rw, rq)

		return false, 0
	}

	return true, 0
}

func (s *SouinTraefikMiddleware) handleResponse(reqCtx uint32, req api.Request, resp api.Response, isError bool) {
	// s.SouinBaseHandler.Configuration.GetLogger().Debug("handleResponse")
	return
}
