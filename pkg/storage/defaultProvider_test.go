package storage

import (
	"testing"
	"time"

	"github.com/darkweak/souin/tests"
)

func TestDefault_DeleteMany(t *testing.T) {
	cases := map[string]struct {
		key     string
		deleted []string
		kept    []string
	}{
		"regexp matches unanchored": {
			key:     "GET-domain.com-/blog",
			deleted: []string{"GET-domain.com-/blog", "GET-domain.com-/blog/a", "IDX_GET-domain.com-/blog"},
			kept:    []string{"GET-domain.com-/news"},
		},
		"dollar ends the key": {
			key:     "GET-domain.com-/blog$",
			deleted: []string{"GET-domain.com-/blog", "IDX_GET-domain.com-/blog"},
			kept:    []string{"GET-domain.com-/blog/a", "GET-domain.com-/blogger", "GET-domain.com-/blog?page=2"},
		},
		"literal key that is not a self-matching regexp": {
			key:     "GET-domain.com-/a+b",
			deleted: []string{"GET-domain.com-/a+b", "GET-domain.com-/ab"},
			kept:    []string{"GET-domain.com-/a+b/c", "GET-domain.com-/a+bc"},
		},
		"literal key that is not a valid regexp": {
			key:     "GET-domain.com-/a(",
			deleted: []string{"GET-domain.com-/a("},
			kept:    []string{"GET-domain.com-/a(b", "GET-domain.com-/a"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			provider, err := Factory(tests.MockConfiguration(tests.BaseConfiguration))
			if err != nil {
				t.Fatalf("Impossible to create the storer, %v given", err)
			}

			for _, k := range append(append([]string{}, tc.deleted...), tc.kept...) {
				if err := provider.Set(k, []byte("value"), time.Minute); err != nil {
					t.Fatalf("Impossible to store the key %s, %v given", k, err)
				}
			}

			provider.DeleteMany(tc.key)

			for _, k := range tc.deleted {
				if len(provider.Get(k)) != 0 {
					t.Errorf("The key %s should be deleted by DeleteMany(%q)", k, tc.key)
				}
			}
			for _, k := range tc.kept {
				if len(provider.Get(k)) == 0 {
					t.Errorf("The key %s should not be deleted by DeleteMany(%q)", k, tc.key)
				}
			}
		})
	}
}
