package main

import (
	"log"

	souin_skipper "github.com/darkweak/souin/plugins/skipper"
	"github.com/zalando/skipper"
	"github.com/zalando/skipper/filters"
)

func main() {
	log.Fatal(
		skipper.Run(skipper.Options{
			Address:       ":9090",
			RoutesFile:    "example.yaml",
			CustomFilters: []filters.Spec{souin_skipper.NewSouinFilter()}},
		),
	)
}
