package urlpattern

import "unicode/utf8"

// Adapted from the regexp package (/ added to the list of special chars): https://cs.opensource.google/go/go/+/refs/tags/go1.23.0:src/regexp/regexp.go;l=705-747

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found at https://go.dev/LICENSE.

// Bitmap used by func special to check whether a character needs to be escaped.
var specialRegexpBytes [16]byte
var specialPatternBytes [16]byte

// specialRegexp reports whether byte b needs to be escaped by QuoteMeta.
func specialRegexp(b byte) bool {
	return b < utf8.RuneSelf && specialRegexpBytes[b%16]&(1<<(b/16)) != 0
}

// specialPattern reports whether byte b needs to be escaped by QuoteMeta.
func specialPattern(b byte) bool {
	return b < utf8.RuneSelf && specialPatternBytes[b%16]&(1<<(b/16)) != 0
}

func init() {
	for _, b := range []byte(`\.+*?()|[]{}^$/`) {
		specialRegexpBytes[b%16] |= 1 << (b / 16)
	}
	for _, b := range []byte(`\+*?(){}:`) {
		specialPatternBytes[b%16] |= 1 << (b / 16)
	}
}

// https://urlpattern.spec.whatwg.org/#escape-a-regexp-string
func escapeRegexpString(s string) string {
	// A byte loop is correct because all metacharacters are ASCII.
	var i int
	for i = 0; i < len(s); i++ { //nolint:intrange // i is reused below
		if specialRegexp(s[i]) {
			break
		}
	}
	// No meta characters found, so return original string.
	if i >= len(s) {
		return s
	}

	b := make([]byte, 2*len(s)-i)
	copy(b, s[:i])
	j := i
	for ; i < len(s); i++ {
		if specialRegexp(s[i]) {
			b[j] = '\\'
			j++
		}
		b[j] = s[i]
		j++
	}
	return string(b[:j])
}

// https://urlpattern.spec.whatwg.org/#escape-a-pattern-string
func escapePatternString(s string) string {
	// A byte loop is correct because all metacharacters are ASCII.
	var i int
	for i = 0; i < len(s); i++ { //nolint:intrange // i is reused below
		if specialPattern(s[i]) {
			break
		}
	}
	// No meta characters found, so return original string.
	if i >= len(s) {
		return s
	}

	b := make([]byte, 2*len(s)-i)
	copy(b, s[:i])
	j := i
	for ; i < len(s); i++ {
		if specialPattern(s[i]) {
			b[j] = '\\'
			j++
		}
		b[j] = s[i]
		j++
	}
	return string(b[:j])
}
