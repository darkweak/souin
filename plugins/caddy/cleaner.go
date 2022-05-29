package httpcache

import (
	"fmt"
	"sync"
)

const stored_providers_key = "STORED_PROVIDERS_KEY"

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

func (s *SouinCaddyPlugin) Cleanup() error {
	fmt.Println("Cleanup...")
	td := []interface{}{}
	sp, _ := up.LoadOrStore(stored_providers_key, newStorageProvider())
	stored_providers := sp.(*storage_providers)
	fmt.Println(stored_providers.list)
	up.Range(func(key, value interface{}) bool {
		if key != stored_providers_key {
			if !stored_providers.list[key] {
				td = append(td, key)
			}
		}

		return true
	})

	for _, v := range td {
		fmt.Printf("Cleaning %v\n", v)
		up.Delete(v)
	}

	return nil
}
