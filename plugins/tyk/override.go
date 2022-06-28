package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"unsafe"

	"github.com/TykTechnologies/tyk/apidef"
	"github.com/darkweak/souin/api"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/cache/surrogate"
	"github.com/darkweak/souin/cache/types"
	"github.com/darkweak/souin/cache/ykeys"
	"github.com/darkweak/souin/plugins"
	"github.com/darkweak/souin/rfc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type souinAPIDefinition struct {
	*apidef.APIDefinition
	Souin Configuration `json:"souin,omitempty"`
}

func parseSouinDefinition(b []byte) *souinAPIDefinition {
	def := souinAPIDefinition{}
	if err := json.Unmarshal(b, &def); err != nil {
		fmt.Println("[RPC] --> Couldn't unmarshal api configuration: ", err)
	}
	cfg := zap.Config{
		Encoding:         "json",
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			CallerKey:    "caller",
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}
	logger, _ := cfg.Build()
	def.Souin.logger = logger
	def.Souin.SetLogger(logger)

	return &def
}

func apiDefinitionRetriever(currentCtx interface{}) *apidef.APIDefinition {
	contextValues := reflect.ValueOf(currentCtx).Elem()
	contextKeys := reflect.TypeOf(currentCtx).Elem()

	if contextKeys.Kind() == reflect.Struct {
		for i := 0; i < contextValues.NumField(); i++ {
			rv := contextValues.Field(i)
			reflectValue := reflect.NewAt(rv.Type(), unsafe.Pointer(rv.UnsafeAddr())).Elem().Interface()

			reflectField := contextKeys.Field(i)

			if reflectField.Name == "Context" {
				apiDefinitionRetriever(reflectValue)
			} else if fmt.Sprintf("%T", reflectValue) == "*apidef.APIDefinition" {
				apidefinition := apidef.APIDefinition{}
				b, _ := json.Marshal(reflectValue)
				e := json.Unmarshal(b, &apidefinition)
				if e == nil {
					return &apidefinition
				}
			}
		}
	}

	return nil
}

func fromDir(dir string) map[string]*souinInstance {
	c := make(map[string]*souinInstance)
	var provider types.AbstractProviderInterface
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, path := range paths {
		f, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}
		def := parseSouinDefinition(f)
		config := &def.Souin
		retriever := plugins.DefaultSouinPluginInitializerFromConfiguration(config)
		if provider != nil {
			retriever.Provider = provider
		} else {
			provider = retriever.Provider
		}
		retriever.Transport = rfc.NewTransport(provider, ykeys.InitializeYKeys(config.GetYkeys()), surrogate.InitializeSurrogate(config))
		fmt.Println(`message="Souin configuration is now loaded."`)

		c[def.APIID] = &souinInstance{
			MapHandler:        api.GenerateHandlerMap(config, retriever.GetTransport()),
			RequestCoalescing: coalescing.Initialize(),
			Retriever:         retriever,
			Configuration:     config,
		}
	}
	return c
}
