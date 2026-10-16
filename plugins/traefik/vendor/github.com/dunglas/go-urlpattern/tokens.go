package urlpattern

// https://wicg.github.io/urlpattern/#tokens
type token struct {
	tType tokenType
	index int
	value string
}

type tokenType uint8

const (
	// tokenOpen represents a U+007B ({) code point.
	tokenOpen tokenType = iota
	// tokenClose represents a U+007D (}) code point.
	tokenClose
	// tokenRegexp represents a string of the form "(<regular expression>)". The regular expression is required to consist of only ASCII code points.
	tokenRegexp
	// tokenName a string of the form ":<name>". The name value is restricted to code points that are consistent with JavaScript identifiers.
	tokenName
	// tokenChar represents a valid pattern code point without any special syntactical meaning.
	tokenChar
	// tokenEscapedChar represents a code point escaped using a backslash like "\<char>".
	tokenEscapedChar
	// tokenOtherModifier represents a matching group modifier that is either the U+003F (?) or U+002B (+) code points.
	tokenOtherModifier
	// tokenAsterisk represents a U+002A (*) code point that can be either a wildcard matching group or a matching group modifier.
	tokenAsterisk
	// tokenEnd represents the end of the pattern string.
	tokenEnd
	// tokenInvalidChar represents a code point that is invalid in the pattern. This could be because of the code point value itself or due to its location within the pattern relative to other syntactic elements.
	tokenInvalidChar
)
