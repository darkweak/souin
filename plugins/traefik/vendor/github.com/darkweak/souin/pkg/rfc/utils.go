package rfc

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
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

// HTTP dates come in exactly three shapes (RFC 9110 section 5.6.7) and
// nothing else may be understood: a recipient that guesses at a malformed
// date would keep serving a response the origin meant to expire. Go's own
// date parser is lenient about field widths and zone names, so each shape is
// matched strictly before being handed over to it.
var httpDateFormats = []struct {
	pattern *regexp.Regexp
	layout  string
}{
	{
		// IMF-fixdate, the preferred format.
		pattern: regexp.MustCompile(`^(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun), \d{2} (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{4} \d{2}:\d{2}:\d{2} GMT$`),
		layout:  http.TimeFormat,
	},
	{
		// The obsolete RFC 850 format.
		pattern: regexp.MustCompile(`^(?:Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday), \d{2}-(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)-\d{2} \d{2}:\d{2}:\d{2} GMT$`),
		layout:  time.RFC850,
	},
	{
		// The obsolete asctime() format.
		pattern: regexp.MustCompile(`^(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?:\d{2}| \d) \d{2}:\d{2}:\d{2} \d{4}$`),
		layout:  time.ANSIC,
	},
}

// knownTimeZoneAbbreviations are the zone names an HTTP date may carry. They
// are spelled in upper case, unlike the month and weekday names around them,
// which is what makes normalising the case of a date field worth doing at
// all.
var knownTimeZoneAbbreviations = map[string]bool{
	"GMT": true, "UTC": true, "UT": true, "Z": true,
	"EST": true, "EDT": true, "CST": true, "CDT": true,
	"MST": true, "MDT": true, "PST": true, "PDT": true,
}

// ParseHTTPDate parses an HTTP date field value. An origin spelling the
// month, weekday or zone in the wrong case is still understood, since the
// alternative is to treat the response as having no expiry at all.
func ParseHTTPDate(value string) (time.Time, error) {
	if parsed, err := parseStrictHTTPDate(value); err == nil {
		return parsed, nil
	}

	return parseStrictHTTPDate(normalizeHTTPDateCase(value))
}

func parseStrictHTTPDate(value string) (time.Time, error) {
	for _, format := range httpDateFormats {
		if format.pattern.MatchString(value) {
			return time.Parse(format.layout, value)
		}
	}

	return time.Time{}, fmt.Errorf("%q is not an HTTP date", value)
}

// normalizeHTTPDateCase rewrites every alphabetic run of an HTTP date to the
// casing the date formats expect: upper case for a zone name, capitalized for
// anything else.
func normalizeHTTPDateCase(value string) string {
	var normalized strings.Builder
	normalized.Grow(len(value))

	for start := 0; start < len(value); {
		c := value[start]
		if !isAlpha(c) {
			normalized.WriteByte(c)
			start++

			continue
		}

		end := start
		for end < len(value) && isAlpha(value[end]) {
			end++
		}

		word := value[start:end]
		if knownTimeZoneAbbreviations[strings.ToUpper(word)] {
			normalized.WriteString(strings.ToUpper(word))
		} else {
			normalized.WriteString(strings.ToUpper(word[:1]) + strings.ToLower(word[1:]))
		}

		start = end
	}

	return normalized.String()
}

func isAlpha(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
