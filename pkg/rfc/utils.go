package rfc

import (
	"net/http"
	"strings"
)

// HeaderAllCommaSepValues returns all comma-separated values (each
// with whitespace trimmed) for a given header name. According to
// Section 4.2 of the HTTP/1.1 spec
// (http://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2),
// values from multiple occurrences of a header should be concatenated, if
// the header's value is a comma-separated list.
func HeaderAllCommaSepValues(headers http.Header, headerName string) []string {
	var vals []string
	for _, val := range headers[http.CanonicalHeaderKey(headerName)] {
		fields := strings.Split(val, ",")
		for i, f := range fields {
			trimmedField := strings.TrimSpace(f)
			fields[i] = trimmedField
		}
		vals = append(vals, fields...)
	}
	return vals
}

func HeaderAllCommaSepValuesString(headers http.Header, headerName string) string {
	valsArray := HeaderAllCommaSepValues(headers, headerName)
	return strings.Join(valsArray, ", ")
}

// HasUnqualifiedNoCache reports whether a Cache-Control value contains the
// no-cache directive without field names. Only that form requires the stored
// response to be revalidated before each reuse; the qualified form
// (no-cache="field") only prevents the listed fields from being reused
// (RFC 9111 §5.2.2.4).
func HasUnqualifiedNoCache(cacheControl string) bool {
	inQuote := false
	start := 0
	for i := 0; i <= len(cacheControl); i++ {
		if i < len(cacheControl) {
			switch c := cacheControl[i]; {
			case inQuote && c == '\\':
				i++
				continue
			case c == '"':
				inQuote = !inQuote
				continue
			case c != ',' || inQuote:
				continue
			}
		}
		if strings.EqualFold(strings.TrimSpace(cacheControl[start:i]), "no-cache") {
			return true
		}
		start = i + 1
	}

	return false
}
