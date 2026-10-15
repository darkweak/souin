package rfc

import (
	"strconv"
	"strings"
)

// integerCacheDirectives are the cache directives RFC 9213 defines as
// carrying an Integer. A CDN-Cache-Control giving them anything else is not a
// valid Structured Field and has to be ignored as a whole.
var integerCacheDirectives = map[string]bool{
	"max-age":                true,
	"s-maxage":               true,
	"stale-while-revalidate": true,
	"stale-if-error":         true,
}

// IsValidCacheControlDictionary reports whether value is a well formed RFC
// 8941 Dictionary of cache directives. Headers defined as Structured Fields,
// such as the `CDN-Cache-Control` of RFC 9213, must be ignored entirely when
// they fail to parse, rather than partially honoured.
func IsValidCacheControlDictionary(value string) bool {
	members, ok := splitDictionaryMembers(value)
	if !ok {
		return false
	}

	for _, member := range members {
		key, item, hasItem := strings.Cut(member, "=")
		// Directive names are matched without regard to case, so a key that
		// only differs from a known one by its casing still describes it.
		key = strings.ToLower(strings.TrimSpace(key))
		if !isDictionaryKey(key) {
			return false
		}

		if !hasItem {
			continue
		}

		// Parameters are appended to the item and play no part in the type of
		// the item itself.
		item, _, _ = strings.Cut(item, ";")
		item = strings.TrimSpace(item)

		if integerCacheDirectives[key] {
			if _, err := strconv.Atoi(item); err != nil {
				return false
			}

			continue
		}

		if !isBareItem(item) {
			return false
		}
	}

	return true
}

// splitDictionaryMembers splits on the commas separating dictionary members,
// leaving the ones inside a quoted string alone. It reports whether the value
// holds at least one member and no unterminated string.
func splitDictionaryMembers(value string) ([]string, bool) {
	members := []string{}
	quoted := false
	escaped := false
	current := strings.Builder{}

	for _, c := range value {
		switch {
		case escaped:
			escaped = false
		case quoted && c == '\\':
			escaped = true
		case c == '"':
			quoted = !quoted
		case c == ',' && !quoted:
			members = append(members, current.String())
			current.Reset()

			continue
		}

		current.WriteRune(c)
	}

	if quoted || escaped {
		return nil, false
	}

	members = append(members, current.String())

	for _, member := range members {
		if strings.TrimSpace(member) == "" {
			return nil, false
		}
	}

	return members, true
}

func isDictionaryKey(key string) bool {
	if key == "" {
		return false
	}

	for i, c := range key {
		switch {
		case c >= 'a' && c <= 'z', c == '*':
		case i > 0 && (c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.'):
		default:
			return false
		}
	}

	return true
}

// isBareItem reports whether item is one of the RFC 8941 bare item types: an
// integer or decimal, a string, a token, a byte sequence or a boolean.
func isBareItem(item string) bool {
	if item == "" {
		return false
	}

	switch item[0] {
	case '"':
		return len(item) > 1 && strings.HasSuffix(item, `"`)
	case ':':
		return len(item) > 1 && strings.HasSuffix(item, ":")
	case '?':
		return item == "?0" || item == "?1"
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		if _, err := strconv.ParseFloat(item, 64); err != nil {
			return false
		}

		return true
	}

	return isToken(item)
}

func isToken(item string) bool {
	first := item[0]
	if !(first >= 'a' && first <= 'z' || first >= 'A' && first <= 'Z' || first == '*') {
		return false
	}

	return !strings.ContainsAny(item, " \t\"(),/:;<=>?@[\\]{}")
}
