package main

import (
	"encoding/json"
	"fmt"
	"github.com/TykTechnologies/tyk/apidef"
	"io/ioutil"
	"path/filepath"
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

func fromDir(dir string) map[string]Configuration {
	c := make(map[string]Configuration)
	paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, path := range paths {
		fmt.Println("Loading API Specification from ", path)
		f, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Println("Couldn't open api configuration file: ", err)
			continue
		}
		def := parseSouinDefinition(f)
		c[def.APIID] = def.Souin
	}
	return c
}
