package storage

import (
	"strings"

	"github.com/darkweak/souin/configurationtypes"
	"github.com/darkweak/souin/pkg/storage/types"
	"github.com/darkweak/storages/core"
)

// NewStorages returns the storers the Traefik plugin registered for this
// configuration (see plugins/traefik/storages.go), ordered by the storers
// directive. It mirrors the storers election of the upstream middleware and
// falls back on the default in-memory storer.
func NewStorages(c configurationtypes.AbstractConfigurationInterface) ([]types.Storer, error) {
	dc := c.GetDefaultCache()
	storers := []types.Storer{}

	for _, uuid := range []string{dc.GetOtter().Uuid, dc.GetSimpleFS().Uuid} {
		if uuid == "" {
			continue
		}

		if st := core.GetRegisteredStorer(uuid); st != nil {
			storers = append(storers, st.(types.Storer))
		}
	}

	storers = reorderStorers(storers, dc.GetStorers())
	if len(storers) > 0 {
		return storers, nil
	}

	memoryStorer, err := Factory(c)
	if err != nil {
		return nil, err
	}

	if st := core.GetRegisteredStorer(types.DefaultStorageName + "-"); st != nil {
		memoryStorer = st.(types.Storer)
	} else {
		core.RegisterStorage(memoryStorer)
	}

	return []types.Storer{memoryStorer}, nil
}

func reorderStorers(storers []types.Storer, expectedStorers []string) []types.Storer {
	if len(expectedStorers) == 0 {
		return storers
	}

	newStorers := make([]types.Storer, 0)
	for _, expectedStorer := range expectedStorers {
		for _, storer := range storers {
			if storer.Name() == strings.ToUpper(expectedStorer) {
				newStorers = append(newStorers, storer)
			}
		}
	}

	return newStorers
}
