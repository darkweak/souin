package rfc

import "testing"

func Test_HasUnqualifiedNoCache(t *testing.T) {
	for _, tc := range []struct {
		cacheControl string
		expected     bool
	}{
		{"no-cache", true},
		{"No-CaChE", true},
		{"max-age=60, no-cache", true},
		{"no-cache, no-cache=\"X-Secret\"", true},
		{"no-cache=\"X-Secret\", no-cache", true},
		{"no-cache=\"X-Secret\"", false},
		{"no-cache=\"X-Secret, X-Other\", max-age=60", false},
		{"no-cache=\"a\\\", no-cache\"", false},
		{"no-cache=X-Secret", false},
		{"max-age=60", false},
		{"no-store", false},
		{"", false},
	} {
		if got := HasUnqualifiedNoCache(tc.cacheControl); got != tc.expected {
			t.Errorf("HasUnqualifiedNoCache(%q) = %v, expected %v", tc.cacheControl, got, tc.expected)
		}
	}
}
