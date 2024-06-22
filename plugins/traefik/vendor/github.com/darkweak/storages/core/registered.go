package core

import "sync"

var registered = sync.Map{}

func RegisterStorage(s Storer) {
	registered.Store(s.Name(), s)
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
