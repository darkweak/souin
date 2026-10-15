package urlpattern

import (
	"errors"
	"strings"
	"unicode"
)

type partType uint8

const (
	// partFixedText represents a simple fixed text string.
	partFixedText partType = iota
	// partRegexp represents a matching group with a custom regular expression.
	partRegexp
	// partSegmentWildcard represents a matching group that matches code points up to the next separator code point. This is typically used for a named group like ":foo" that does not have a custom regular expression.
	partSegmentWildcard
	// partFullWildcard represents a matching group that greedily matches all code points. This is typically used for the "*" wildcard matching group.
	partFullWildcard
)

var (
	ErrEmptyPartName         = errors.New("part's name must not be empty string")
	ErrInvalidModifier       = errors.New(`part's modifier must be "zero-or-more" or "one-or-more"`)
	ErrInvalidPrefixOrSuffix = errors.New("part's prefix is not the empty string or part's suffix is not the empty string")
	ErrInvalidPartName       = errors.New("part's name is not the empty string or null")
)

type partModifier uint8

const (
	// The part does not have a modifier.
	partModifierNone partModifier = iota
	// The part has an optional modifier indicated by the U+003F (?) code point.
	partModifierOptional
	// The part has a "zero or more" modifier indicated by the U+002A (*) code point.
	partModifierZeroOrMore
	// The part has a "one or more" modifier indicated by the U+002B (+) code point.
	partModifierOneOrMore
)

type part struct {
	pType    partType
	value    string
	modifier partModifier
	name     string
	prefix   string
	suffix   string
}

type partList []part

// https://urlpattern.spec.whatwg.org/#generate-a-regular-expression-and-name-list
func (pl partList) generateRegularExpressionAndNameList(options options) (string, []string, error) {
	var result strings.Builder
	nameList := make([]string, 0, len(pl))

	// the v flag doesn't exist in Go
	if options.ignoreCase {
		result.WriteString("(?i)")
	}

	result.WriteString("\\A(?:")

	for _, p := range pl {
		if p.pType == partFixedText {
			if p.modifier == partModifierNone {
				result.WriteString(escapeRegexpString(p.value))
			} else {
				result.WriteString("(?:")
				result.WriteString(escapeRegexpString(p.value))
				result.WriteByte(')')

				if modifierToString := convertModifierToString(p.modifier); modifierToString != 0 {
					result.WriteByte(modifierToString)
				}
			}

			continue
		}

		// Assert: part's name is not the empty string.
		if p.name == "" {
			return "", nil, ErrEmptyPartName
		}

		nameList = append(nameList, p.name)

		var regexpValue string
		switch p.pType {
		case partSegmentWildcard:
			regexpValue = generateSegmentWildcardRegexp(options)
		case partFullWildcard:
			regexpValue = fullWildcardRegexpValue
		default:
			regexpValue = p.value
		}

		if p.prefix == "" && p.suffix == "" {
			switch p.modifier {
			case partModifierNone, partModifierOptional:
				result.WriteByte('(')
				result.WriteString(regexpValue)
				result.WriteByte(')')

				if modifierToString := convertModifierToString(p.modifier); modifierToString != 0 {
					result.WriteByte(modifierToString)
				}

			default:
				result.WriteString("((?:")
				result.WriteString(regexpValue)
				result.WriteByte(')')

				if modifierToString := convertModifierToString(p.modifier); modifierToString != 0 {
					result.WriteByte(modifierToString)
				}

				result.WriteByte(')')
			}

			continue
		}

		if p.modifier == partModifierNone || p.modifier == partModifierOptional {
			result.WriteString("(?:")
			result.WriteString(escapeRegexpString((p.prefix)))
			result.WriteByte('(')
			result.WriteString(regexpValue)
			result.WriteByte(')')
			result.WriteString(escapeRegexpString((p.suffix)))
			result.WriteByte(')')

			if modifierToString := convertModifierToString(p.modifier); modifierToString != 0 {
				result.WriteByte(modifierToString)
			}

			continue
		}

		// Assert: part’s modifier is "zero-or-more" or "one-or-more".
		if p.modifier != partModifierZeroOrMore && p.modifier != partModifierOneOrMore {
			return "", nil, ErrInvalidModifier
		}

		// Assert: part’s prefix is not the empty string or part’s suffix is not the empty string.
		if p.prefix == "" && p.suffix == "" {
			return "", nil, ErrInvalidPrefixOrSuffix
		}

		result.WriteString("(?:")
		result.WriteString(escapeRegexpString(p.prefix))
		result.WriteString("((?:")
		result.WriteString(regexpValue)
		result.WriteString(")(?:")
		result.WriteString(escapeRegexpString(p.suffix))
		result.WriteString(escapeRegexpString(p.prefix))
		result.WriteString("(?:")
		result.WriteString(regexpValue)
		result.WriteString("))*)")
		result.WriteString(escapeRegexpString(p.suffix))
		result.WriteByte(')')
		if p.modifier == partModifierZeroOrMore {
			result.WriteByte('?')
		}
	}

	result.WriteString(")\\z")

	return result.String(), nameList, nil
}

// https://urlpattern.spec.whatwg.org/#generate-a-pattern-string
func (pl partList) generatePatternString(options options) (string, error) {
	var result strings.Builder

	maxIndex := len(pl) - 1
	var previousPart *part
	var nextPart *part
	for index, part := range pl {
		if index > 0 {
			previousPart = &pl[index-1]
		}
		if index < maxIndex {
			nextPart = &pl[index+1]
		} else {
			nextPart = nil
		}

		if part.pType == partFixedText {
			if part.modifier == partModifierNone {
				result.WriteString(escapePatternString(part.value))

				continue
			}

			result.WriteByte('{')
			result.WriteString(escapePatternString(part.value))
			result.WriteByte('}')
			if modifier := convertModifierToString(part.modifier); modifier != 0 {
				result.WriteByte(modifier)
			}

			continue
		}

		customName := !unicode.IsDigit([]rune(part.name)[0])
		needGrouping := part.suffix != "" || (part.prefix != "" && part.prefix != string(options.prefixCodePoint))

		if !needGrouping &&
			customName &&
			part.pType == partSegmentWildcard &&
			part.modifier == partModifierNone &&
			nextPart != nil &&
			nextPart.prefix == "" &&
			nextPart.suffix == "" {
			// A fixed-text part's value can be empty: some encoding callbacks
			// (e.g. hostname/pathname canonicalization) can collapse a non-empty
			// prefix/suffix down to "" for syntax-only input such as "#".
			if nextPart.pType == partFixedText {
				if nextPart.value != "" && isValidNameCodePoint([]rune(nextPart.value)[0], false) {
					needGrouping = true
				}
			} else if unicode.IsDigit([]rune(nextPart.name)[0]) {
				needGrouping = true
			}
		}

		if !needGrouping &&
			part.prefix == "" &&
			previousPart != nil &&
			previousPart.pType == partFixedText &&
			previousPart.value != "" &&
			[]rune(previousPart.value)[len(previousPart.value)-1] == rune(options.prefixCodePoint) {
			needGrouping = true
		}

		// Assert: part’s name is not the empty string or null.
		if part.name == "" {
			return "", ErrInvalidPartName
		}

		if needGrouping {
			result.WriteByte('{')
		}

		result.WriteString(escapePatternString(part.prefix))

		if customName {
			result.WriteByte(':')
			result.WriteString(part.name)
		}

		switch part.pType {
		case partRegexp:
			result.WriteByte('(')
			result.WriteString(part.value)
			result.WriteByte(')')

		case partSegmentWildcard:
			if !customName {
				result.WriteByte('(')
				result.WriteString(generateSegmentWildcardRegexp(options))
				result.WriteByte(')')
			}

		case partFullWildcard:
			if !customName && (previousPart == nil ||
				previousPart.pType == partFixedText ||
				previousPart.modifier != partModifierNone ||
				needGrouping ||
				part.prefix != "") {
				result.WriteByte('*')
			} else {
				result.WriteByte('(')
				result.WriteString(fullWildcardRegexpValue)
				result.WriteByte(')')
			}
		}

		if part.pType == partSegmentWildcard &&
			customName &&
			part.suffix != "" &&
			isValidNameCodePoint([]rune(part.suffix)[0], false) {
			result.WriteByte('\\')
		}

		result.WriteString(escapePatternString(part.suffix))

		if needGrouping {
			result.WriteByte('}')
		}

		if modifierToString := convertModifierToString(part.modifier); modifierToString != 0 {
			result.WriteByte(modifierToString)
		}
	}

	return result.String(), nil
}

// https://urlpattern.spec.whatwg.org/#convert-a-modifier-to-a-string
func convertModifierToString(m partModifier) byte {
	switch m {
	case partModifierZeroOrMore:
		return '*'
	case partModifierOptional:
		return '?'
	case partModifierOneOrMore:
		return '+'
	default:
		return 0
	}
}
