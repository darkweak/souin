// Package plugindemo a demo plugin.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler"
	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
)

func main() {
	var config Config
	fmt.Println(api.LogLevelInfo, "INIT")
	handler.Host.Log(api.LogLevelInfo, "INIT")
	err := json.Unmarshal(handler.Host.GetConfig(), &config)
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
}

// Config the plugin configuration.
type Config struct {
	Headers map[string]string `json:"headers,omitempty"`
}

// Demo a Demo plugin.
type Demo struct {
	headers  map[string]string
	template *template.Template
}

// New created a new Demo plugin.
func New(config Config) (*Demo, error) {
	// if len(config.Headers) == 0 {
	// 	return nil, fmt.Errorf("headers cannot be empty")
	// }

	return &Demo{
		headers:  config.Headers,
		template: template.New("demo").Delims("[[", "]]"),
	}, nil
}

func (a *Demo) handleRequest(req api.Request, resp api.Response) (next bool, reqCtx uint32) {
	for key, value := range a.headers {
		tmpl, err := a.template.Parse(value)
		if err != nil {
			resp.SetStatusCode(http.StatusInternalServerError)
			resp.Body().Write([]byte(err.Error()))
			return false, 0
		}

		writer := &bytes.Buffer{}

		err = tmpl.Execute(writer, req)
		if err != nil {
			resp.SetStatusCode(http.StatusInternalServerError)
			resp.Body().Write([]byte(err.Error()))
			return false, 0
		}

		req.Headers().Set(key, writer.String())
	}

	return true, 0
}
