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
