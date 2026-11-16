package providers

import (
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/storage/types"
)

// setStorer keeps tags as native sets, like the redis and go-redis storages.
// Its DeleteMany matches set keys too, as theirs does: both scan every key
// whatever its type.
type setStorer struct {
	types.Storer
	mu   sync.Mutex
	sets map[string]map[string]struct{}
}

func (s *setStorer) AddToSet(key string, members []string, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sets[key] == nil {
		s.sets[key] = map[string]struct{}{}
	}

	for _, m := range members {
		s.sets[key][m] = struct{}{}
	}

	return nil
}

func (s *setStorer) GetSet(key string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	members := []string{}
	for m := range s.sets[key] {
		members = append(members, m)
	}

	return members
}

func (s *setStorer) WalkSets(prefix string, fn func(key string, members []string) bool) error {
	s.mu.Lock()
	keys := []string{}
	for k := range s.sets {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	s.mu.Unlock()

	for _, k := range keys {
		if !fn(strings.TrimPrefix(k, prefix), s.GetSet(k)) {
			break
		}
	}

	return nil
}

func (s *setStorer) DeleteMany(key string) {
	if rg, err := regexp.Compile(key); err == nil {
		s.mu.Lock()
		for k := range s.sets {
			if rg.MatchString(k) {
				delete(s.sets, k)
			}
		}
		s.mu.Unlock()
	}

	s.Storer.DeleteMany(key)
}

func TestBaseStorage_Clear_SetBackedTags(t *testing.T) {
	bs := mockCommonProvider()
	bs.duration = time.Hour
	fake := &setStorer{Storer: bs.Storage, sets: map[string]map[string]struct{}{}}
	bs.Storage = fake

	bs.storeTag("tagA", "k1")
	bs.storeTag("tagB", "k2")

	if len(fake.sets) != 2 || len(bs.List()) != 2 {
		t.Fatalf("Both tags should be stored as sets before Clear, %v given", bs.List())
	}

	bs.Clear()

	if got := bs.List(); len(got) != 0 {
		t.Errorf("Clear should delete every set-backed surrogate key, %v left", got)
	}

	bs.storeTag("tagA", "k3")

	if got := fake.GetSet(surrogatePrefix + "tagA"); len(got) != 1 || got[0] != "k3" {
		t.Errorf("The storer should keep storing set-backed tags after Clear, %v given", got)
	}
}
