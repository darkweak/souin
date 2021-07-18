package main

import (
	"encoding/json"
	"fmt"
	"github.com/TykTechnologies/tyk/apidef"
	"github.com/darkweak/souin/cache/coalescing"
	"github.com/darkweak/souin/plugins"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"unsafe"
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

func fromDir(dir string) map[string]souinInstance {
	c := make(map[string]souinInstance)
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, path := range paths {
		fmt.Println("Loading API Specification from ", path)
		f, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("Couldn't open api configuration file: ", err)
			continue
		}
		def := parseSouinDefinition(f)
		config := &def.Souin

		c[def.APIID] = souinInstance{
			Retriever:         plugins.DefaultSouinPluginInitializerFromConfiguration(config),
			RequestCoalescing: coalescing.Initialize(),
			Configuration:     config,
		}
	}
	return c
}
