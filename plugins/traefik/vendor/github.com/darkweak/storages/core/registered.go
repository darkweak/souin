package core

import (
	"fmt"
	"sync"
)

var registered = sync.Map{}

func RegisterStorage(s Storer) {
	_ = s.Init()
	registered.Store(fmt.Sprintf("%s-%s", s.Name(), s.Uuid()), s)
}

func GetRegisteredStorer(name string) Storer {
	s, _ := registered.Load(name)
	if s != nil {
		if v, ok := s.(Storer); ok {
			return v
		}
	}

	return nil
}

func ResetRegisteredStorages() {
	registered.Range(func(key, _ any) bool {
		registered.Delete(key)

		return true
	})

	registered = sync.Map{}
}

func GetRegisteredStorers() []Storer {
	storers := make([]Storer, 0)

	registered.Range(func(_, value any) bool {
		if s, ok := value.(Storer); ok {
			storers = append(storers, s)
		}

		return true
	})

	return storers
}
