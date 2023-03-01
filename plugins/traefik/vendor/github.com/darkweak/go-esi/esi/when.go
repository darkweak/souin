package esi

import (
	"net/http"
	"regexp"
	"strings"
)

var (
	unaryNegation = regexp.MustCompile(`!\((\$\((.+)\)|(.+))\)`)
	comparison    = regexp.MustCompile(`(.+)(==|!=|<=|>=|<|>)(.+)`)
	logicalAnd    = regexp.MustCompile(`\((.+?)\)&\((.+?)\)`)
	logicalOr     = regexp.MustCompile(`\((.+?)\)\|\((.+?)\)`)
)

func validateTest(b []byte, req *http.Request) bool {
	if r := unaryNegation.FindSubmatch(b); r != nil {
		return !validateTest(r[1], req)
	} else if r := logicalAnd.FindSubmatch(b); r != nil {
		return validateTest(r[1], req) && validateTest(r[2], req)
	} else if r := logicalOr.FindSubmatch(b); r != nil {
		return validateTest(r[1], req) || validateTest(r[2], req)
	} else if r := comparison.FindSubmatch(b); r != nil {
		r1 := strings.TrimSpace(parseVariables(r[1], req))
		r2 := strings.TrimSpace(parseVariables(r[3], req))
		switch string(r[2]) {
		case "==":
			return r1 == r2
		case "!=":
			return r1 != r2
		case "<":
			return r1 < r2
		case ">":
			return r1 > r2
		case "<=":
			return r1 <= r2
		case ">=":
			return r1 >= r2
		}
	} else {
		vars := interpretedVar.FindSubmatch(b)
		if vars == nil {
			return false
		}

		return parseVariables(vars[0], req) == "true"
	}

	return false
}
