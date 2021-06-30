package main

import (
	"encoding/json"
	"fmt"
	"github.com/TykTechnologies/tyk/apidef"
	"github.com/darkweak/souin/configurationtypes"
	"io"
	"os"
	"path/filepath"
)

type souinAPIDefinition struct {
	*apidef.APIDefinition
	Configuration configurationtypes.AbstractConfigurationInterface
}

func parseSouinDefinition(r io.Reader) *souinAPIDefinition {
	def := &souinAPIDefinition{}
	if err := json.NewDecoder(r).Decode(def); err != nil {
		fmt.Println("[RPC] --> Couldn't unmarshal api configuration: ", err)
	}
	return def
}

func merge(a, b interface{}) interface{} {
	jb, err := json.Marshal(b)
	if err != nil {
		fmt.Println("Marshal error b:", err)
	}
	err = json.Unmarshal(jb, &a)
	if err != nil {
		fmt.Println("Unmarshal error b-a:", err)
	}
	return a
}

func fromDir(dir string) Configuration {
	var c Configuration
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, path := range paths {
		fmt.Println("Loading API Specification from ", path)
		f, err := os.Open(path)
		if err != nil {
			fmt.Println("Couldn't open api configuration file: ", err)
			continue
		}
		merge(&c, parseSouinDefinition(f))
	}
	return c
}
