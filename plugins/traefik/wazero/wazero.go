package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/http-wasm/http-wasm-host-go/handler"
	wasm "github.com/http-wasm/http-wasm-host-go/handler/nethttp"
	"github.com/juliens/wasm-goexport/host"
	"github.com/tetratelabs/wazero"
	"github.com/traefik/traefik/v3/pkg/logs"
	"github.com/traefik/traefik/v3/pkg/middlewares"
)

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
		fmt.Printf("compiling guest module: %#v\n", err)

		return
	}

	// applyCtx, err := plugins.InstantiateHost(ctx, rt, gm, plugins.Settings{})
	// if err != nil {
	// 	fmt.Printf("instantiating host module: %#v\n", err)
	//
	// 	return
	// }

	logger := middlewares.GetLogger(ctx, "souin", "wasm")

	config := wazero.NewModuleConfig().WithSysWalltime()

	opts := []handler.Option{
		handler.ModuleConfig(config),
		handler.Logger(logs.NewWasmLogger(logger)),
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
}
