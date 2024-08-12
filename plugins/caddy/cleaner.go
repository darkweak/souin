package httpcache

import (
	"sync"
)

const stored_providers_key = "STORED_PROVIDERS_KEY"
const coalescing_key = "COALESCING"
const surrogate_key = "SURROGATE"

type storage_providers struct {
	list map[interface{}]bool
	sync.RWMutex
}

func newStorageProvider() *storage_providers {
	return &storage_providers{
		list:    make(map[interface{}]bool),
		RWMutex: sync.RWMutex{},
	}
}

func (s *storage_providers) Add(key interface{}) {
	s.RWMutex.Lock()
	defer s.RWMutex.Unlock()

	s.list[key] = true
}

func (s *SouinCaddyMiddleware) Cleanup() error {
	s.logger.Debug("Cleanup...")
	td := []interface{}{}
	sp, _ := up.LoadOrStore(stored_providers_key, newStorageProvider())
	stored_providers := sp.(*storage_providers)
	up.Range(func(key, _ interface{}) bool {
		if key != stored_providers_key && key != coalescing_key && key != surrogate_key {
			if !stored_providers.list[key] {
				td = append(td, key)
			}
		}

		return true
	})

	for _, v := range td {
		s.logger.Debugf("Cleaning %v\n", v)
		_, _ = up.Delete(v)
	}

	return nil
}
