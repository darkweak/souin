package esi

import "regexp"

const (
	try = "try"
)

var (
	esi     = regexp.MustCompile("<esi:")
	tagname = regexp.MustCompile("^(([a-z]+)|(<!--esi))")

	// closeTry       = regexp.MustCompile("</esi:try>").
)
