package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/http-wasm/http-wasm-guest-tinygo/handler/api"
	"github.com/http-wasm/http-wasm-host-go/handler"
	wasm "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/juliens/wasm-goexport/host"
	"github.com/tetratelabs/wazero"
	"github.com/traefik/traefik/v3/pkg/logs"
	"github.com/traefik/traefik/v3/pkg/middlewares"
)

var _ api.Host = wasmHost{}

type wasmHost struct {
	baseHost api.Host
}

// EnableFeatures implements api.Host.
func (w wasmHost) EnableFeatures(api.Features) api.Features {
	return api.FeatureBufferResponse
}

// GetConfig implements api.Host.
func (w wasmHost) GetConfig() []byte {
	b, _ := json.Marshal(map[string]interface{}{
		"default_cache": map[string]interface{}{
			"ttl": "120s",
		},
	})

	return b
}

// Log implements api.Host.
func (w wasmHost) Log(l api.LogLevel, s string) {
	w.baseHost.Log(l, s)
}

// LogEnabled implements api.Host.
func (w wasmHost) LogEnabled(l api.LogLevel) bool {
	return w.baseHost.LogEnabled(l)
}

type server struct{}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("something"))
}

func main() {
	ctx := context.Background()
	code, err := os.ReadFile("../wasm/plugin.wasm")
	if err != nil {
		fmt.Printf("loading binary: %#v\n", err)

		return
	}

	rt := host.NewRuntime(wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCompilationCache(wazero.NewCompilationCache())))

	_, err = rt.CompileModule(ctx, code)
	if err != nil {
		fmt.Println(fmt.Errorf("compiling guest module: %w", err))
	}

	// applyCtx, err := plugins.InstantiateHost(ctx, rt, guestModule, plugins.Settings{})
	// if err != nil {
	// 	fmt.Println(fmt.Errorf("instantiating host module: %w", err))
	// }

	logger := middlewares.GetLogger(ctx, "souin", "wasm")

	config := wazero.NewModuleConfig().WithSysWalltime()
	opts := []handler.Option{
		handler.ModuleConfig(config),
		handler.Logger(logs.NewWasmLogger(logger)),
	}

	data, err := json.Marshal(map[string]interface{}{
		"default_cache": map[string]interface{}{
			"ttl": "120s",
		},
	})
	if err != nil {
		fmt.Println(fmt.Errorf("marshaling config: %w", err))
	}

	opts = append(opts, handler.GuestConfig(data))

	opts = append(opts, handler.Runtime(func(ctx context.Context) (wazero.Runtime, error) {
		return rt, nil
	}))

	mw, err := wasm.NewMiddleware(ctx, code, opts...)
	if err != nil {
		fmt.Println(fmt.Errorf("creating middleware: %w", err))
	}

	h := mw.NewHandler(ctx, &server{})

	// applyCtx, err := plugins.InstantiateHost(ctx, rt, gm, plugins.Settings{})
	// if err != nil {
	// 	fmt.Printf("instantiating host module: %#v\n", err)
	//
	// 	return
	// }
	/*
		logger := middlewares.GetLogger(ctx, "souin", "wasm")

		config := wazero.NewModuleConfig().WithSysWalltime()

		opts := []handler.Option{
			handler.ModuleConfig(config),
			handler.Logger(logs.NewWasmLogger(logger)),
			handler.GuestConfig(wasmHost{}.GetConfig()),
		}

		opts = append(opts, handler.Runtime(func(ctx context.Context) (wazero.Runtime, error) {
			return rt, nil
		}))

		mw, err := wasm.NewMiddleware(ctx, code, opts...)
		if err != nil {
			fmt.Printf("creating middleware: %#v\n", err)

			return
		}

		h := mw.NewHandler(ctx, &server{})
		fmt.Println("Good")
		fmt.Printf("%#v\n", h)
	*/

	http.ListenAndServe(":80", h)
}
