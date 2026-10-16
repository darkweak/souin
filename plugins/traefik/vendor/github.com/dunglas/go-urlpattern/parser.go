package urlpattern

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/nlnwa/whatwg-url/canonicalizer"
	"github.com/nlnwa/whatwg-url/url"
)

// https://urlpattern.spec.whatwg.org/#full-wildcard-regexp-value
const fullWildcardRegexpValue = ".*"

// DefaultPorts maps a protocol scheme to its default port string.
//
// Callers may add entries for additional schemes, but only during program
// initialization and before any concurrent use of this package: as with
// any Go map, concurrent read/write is not safe.
//
// This is an experimental affordance and the symbol may change or be
// removed in a future release.
//
// TODO: there is nothing in the Go stdlib to find the default port
// associated with a protocol. Only the specialSchemeSet entries are
// populated for now; the list can be completed using
// https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers.
var DefaultPorts = map[string]string{
	"http":  "80",
	"https": "443",
	"ws":    "80",
	"wss":   "443",
	"ftp":   "21",
}

var urlParser = url.NewParser()
var hostnameParser = canonicalizer.New(canonicalizer.WithDefaultScheme("http"))

var (
	ErrNonEmptySuffix      = errors.New("suffix must be the empty string")
	ErrBadParserIndex      = errors.New("parser's index must be less than parser's token list size")
	ErrDuplicatePartName   = errors.New("duplicate name")
	ErrRequiredToken       = errors.New("missing required token")
	ErrInvalidIPv6Hostname = errors.New("invalid IPv6 hostname")
	ErrInvalidPort         = errors.New("invalid port")
)

var errInvalidHostname = errors.New("invalid hostname")

// https://urlpattern.spec.whatwg.org/#encoding-callback
type encodingCallback func(string) (string, error)

// https://urlpattern.spec.whatwg.org/#parse-a-pattern-string
func parsePatternString(input string, options options, encodingCallback encodingCallback) (partList, error) {
	tl, err := tokenize(input, tokenizePolicyStrict)
	if err != nil {
		return nil, err
	}

	p := patternParser{
		encodingCallback:      encodingCallback,
		segmentWildcardRegexp: generateSegmentWildcardRegexp(options),
		tokenList:             tl,
	}

	tls := len(tl)

	for p.index < tls {
		charToken, err := p.tryConsumeToken(tokenChar)
		if err != nil {
			return nil, err
		}

		nameToken, err := p.tryConsumeToken(tokenName)
		if err != nil {
			return nil, err
		}

		regexpOrWildcardToken, err := p.tryConsumeRegexpOrWildcardToken(nameToken)
		if err != nil {
			return nil, err
		}

		if nameToken != nil || regexpOrWildcardToken != nil {
			prefix := ""
			if charToken != nil {
				prefix = charToken.value
			}

			if prefix != "" && prefix != string(options.prefixCodePoint) {
				p.pendingFixedValue += prefix
				prefix = ""
			}

			if err := p.maybeAddPartFromPendingFixedValue(); err != nil {
				return nil, err
			}

			modifierToken, err := p.tryConsumeModifierToken()
			if err != nil {
				return nil, err
			}
			if err := p.addPart(prefix, nameToken, regexpOrWildcardToken, "", modifierToken); err != nil {
				return nil, err
			}

			continue
		}

		fixedToken := charToken
		if fixedToken == nil {
			fixedToken, err = p.tryConsumeToken(tokenEscapedChar)
			if err != nil {
				return nil, err
			}
		}
		if fixedToken != nil {
			p.pendingFixedValue += fixedToken.value

			continue
		}

		openToken, err := p.tryConsumeToken(tokenOpen)
		if err != nil {
			return nil, err
		}

		if openToken != nil {
			prefix, err := p.consumeText()
			if err != nil {
				return nil, err
			}

			nameToken, err := p.tryConsumeToken(tokenName)
			if err != nil {
				return nil, err
			}

			regexpOrWildcardToken, err := p.tryConsumeRegexpOrWildcardToken(nameToken)
			if err != nil {
				return nil, err
			}

			suffix, err := p.consumeText()
			if err != nil {
				return nil, err
			}

			if err := p.consumeRequiredToken(tokenClose); err != nil {
				return nil, fmt.Errorf("missing close token: %w", err)
			}

			modifierToken, err := p.tryConsumeModifierToken()
			if err != nil {
				return nil, err
			}

			if err := p.addPart(prefix, nameToken, regexpOrWildcardToken, suffix, modifierToken); err != nil {
				return nil, err
			}

			continue
		}

		if err := p.maybeAddPartFromPendingFixedValue(); err != nil {
			return nil, err
		}

		if err := p.consumeRequiredToken(tokenEnd); err != nil {
			return nil, fmt.Errorf("missing end token: %w", err)
		}
	}

	return p.partList, nil
}

type patternParser struct {
	tokenList             []token
	encodingCallback      encodingCallback
	segmentWildcardRegexp string
	partList              partList
	seenNames             map[string]struct{}
	pendingFixedValue     string
	index                 int
	nextNumericName       float64
}

// https://urlpattern.spec.whatwg.org/#try-to-consume-a-token
func (p *patternParser) tryConsumeToken(tokenType tokenType) (*token, error) {
	// Assert: parser’s index is less than parser’s token list size.
	if p.index >= len(p.tokenList) {
		return nil, ErrBadParserIndex
	}

	nextToken := p.tokenList[p.index]
	if nextToken.tType != tokenType {
		return nil, nil
	}

	p.index++

	return &nextToken, nil
}

// https://urlpattern.spec.whatwg.org/#try-to-consume-a-regexp-or-wildcard-token
func (p *patternParser) tryConsumeRegexpOrWildcardToken(nameToken *token) (*token, error) {
	token, err := p.tryConsumeToken(tokenRegexp)
	if err != nil {
		return nil, err
	}
	if nameToken == nil && token == nil {
		token, err = p.tryConsumeToken(tokenAsterisk)
		if err != nil {
			return nil, err
		}
	}

	return token, nil
}

// https://urlpattern.spec.whatwg.org/#maybe-add-a-part-from-the-pending-fixed-value
func (p *patternParser) maybeAddPartFromPendingFixedValue() error {
	if p.pendingFixedValue == "" {
		return nil
	}

	encodedValue, err := p.encodingCallback(p.pendingFixedValue)
	if err != nil {
		return err
	}

	p.pendingFixedValue = ""

	part := part{pType: partFixedText, value: encodedValue, modifier: partModifierNone}
	p.partList = append(p.partList, part)

	return nil
}

// https://urlpattern.spec.whatwg.org/#try-to-consume-a-modifier-token
func (p *patternParser) tryConsumeModifierToken() (*token, error) {
	token, err := p.tryConsumeToken(tokenOtherModifier)
	if err != nil {
		return nil, err
	}
	if token != nil {
		return token, nil
	}

	return p.tryConsumeToken(tokenAsterisk)
}

// https://urlpattern.spec.whatwg.org/#add-a-part
func (p *patternParser) addPart(prefix string, nameToken *token, regexpOrWildcardToken *token, suffix string, modifierToken *token) error {
	modifier := partModifierNone
	if modifierToken != nil {
		switch modifierToken.value {
		case "?":
			modifier = partModifierOptional
		case "*":
			modifier = partModifierZeroOrMore
		case "+":
			modifier = partModifierOneOrMore
		}
	}

	if nameToken == nil && regexpOrWildcardToken == nil && modifier == partModifierNone {
		p.pendingFixedValue += prefix

		return nil
	}

	if err := p.maybeAddPartFromPendingFixedValue(); err != nil {
		return err
	}

	if nameToken == nil && regexpOrWildcardToken == nil {
		// Assert: suffix is the empty string.
		if suffix != "" {
			return ErrNonEmptySuffix
		}

		if prefix == "" {
			return nil
		}

		encodedValue, err := p.encodingCallback(prefix)
		if err != nil {
			return err
		}

		part := part{pType: partFixedText, value: encodedValue, modifier: modifier}
		p.partList = append(p.partList, part)

		return nil
	}

	var regexpValue string
	switch {
	case regexpOrWildcardToken == nil:
		regexpValue = p.segmentWildcardRegexp
	case regexpOrWildcardToken.tType == tokenAsterisk:
		regexpValue = fullWildcardRegexpValue
	default:
		regexpValue = regexpOrWildcardToken.value
	}

	pType := partRegexp
	switch regexpValue {
	case p.segmentWildcardRegexp:
		pType = partSegmentWildcard
		regexpValue = ""
	case fullWildcardRegexpValue:
		pType = partFullWildcard
		regexpValue = ""
	}

	name := ""
	if nameToken != nil {
		name = nameToken.value
	} else if regexpOrWildcardToken != nil {
		name = strconv.FormatFloat(p.nextNumericName, 'f', -1, 64)
		p.nextNumericName++
	}

	// https://urlpattern.spec.whatwg.org/#is-a-duplicate-name
	if _, seen := p.seenNames[name]; seen {
		return ErrDuplicatePartName
	}

	encodedPrefix, err := p.encodingCallback(prefix)
	if err != nil {
		return err
	}

	encodedSuffix, err := p.encodingCallback(suffix)
	if err != nil {
		return err
	}

	if p.seenNames == nil {
		p.seenNames = make(map[string]struct{})
	}
	p.seenNames[name] = struct{}{}

	part := part{pType: pType, value: regexpValue, modifier: modifier, name: name, prefix: encodedPrefix, suffix: encodedSuffix}
	p.partList = append(p.partList, part)

	return nil
}

// https://urlpattern.spec.whatwg.org/#consume-text
func (p *patternParser) consumeText() (string, error) {
	var result strings.Builder
	for {
		token, err := p.tryConsumeToken(tokenChar)
		if err != nil {
			return "", err
		}
		if token == nil {
			token, err = p.tryConsumeToken(tokenEscapedChar)
			if err != nil {
				return "", err
			}
		}
		if token == nil {
			break
		}
		result.WriteString(token.value)
	}

	return result.String(), nil
}

// https://urlpattern.spec.whatwg.org/#consume-a-required-token
func (p *patternParser) consumeRequiredToken(tokenType tokenType) error {
	result, err := p.tryConsumeToken(tokenType)
	if err != nil {
		return err
	}
	if result == nil {
		return ErrRequiredToken
	}

	return nil
}

// https://urlpattern.spec.whatwg.org/#generate-a-segment-wildcard-regexp
func generateSegmentWildcardRegexp(options options) string {
	return "[^" + escapeRegexpString(string(options.delimiterCodePoint)) + "]+?"
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-protocol
func canonicalizeProtocol(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	dummyURL, err := urlParser.Parse(value + "://dummy.test")
	if err != nil {
		return "", err
	}

	return dummyURL.Scheme(), nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-username
func canonicalizeUsername(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	return urlParser.PercentEncodeString(value, url.UserInfoPercentEncodeSet), nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-password
func canonicalizePassword(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	return urlParser.PercentEncodeString(value, url.UserInfoPercentEncodeSet), nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-hostname
// https://github.com/whatwg/urlpattern/issues/220#issuecomment-2074613501
func canonicalizeHostname(hostnameValue, protocolValue string) (string, error) {
	if hostnameValue == "" {
		return hostnameValue, nil
	}

	// Non-IPv6 hostnames must not contain ':': without this guard, the URL
	// parser would split "host:port" into host and port, silently accepting
	// patterns like "bad:hostname" as a plain hostname.
	if hostnameValue[0] != '[' {
		for _, c := range hostnameValue {
			if c == ':' {
				return "", errInvalidHostname
			}
		}
	}

	var (
		u   *url.Url
		err error
	)

	if protocolValue == "" {
		u = hostnameParser.NewUrl()
	} else {
		u, err = hostnameParser.Parse(protocolValue + "://dummy.test")
		if err != nil {
			return "", err
		}
	}

	u, err = hostnameParser.BasicParser(hostnameValue, nil, u, url.StateHostname)
	if err != nil {
		return "", err
	}

	// The underlying URL parser silently drops forbidden host code points
	// (e.g. "#") instead of failing, turning a non-empty input into an empty
	// hostname. That empty result later reaches partList.generatePatternString
	// as a fixed-text part, which assumes non-empty values and panics.
	if hostname := u.Hostname(); hostname != "" {
		return hostname, nil
	}

	return "", errInvalidHostname
}

// https://github.com/whatwg/urlpattern/issues/220#issuecomment-2074613501
func canonicalizeDomainName(value string) (string, error) {
	return canonicalizeHostname(value, "https")
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-port
func canonicalizePort(portValue, protocolValue string) (string, error) {
	if portValue == "" {
		return portValue, nil
	}

	// The WHATWG port state strips ASCII tab / LF / CR before examining the
	// first code point, so reject inputs whose first significant byte is not
	// an ASCII digit (e.g. "invalid80"). Without this the URL library returns
	// an empty port instead of failing.
	firstDigit := false
	for i := range len(portValue) {
		c := portValue[i]
		if c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		firstDigit = c >= '0' && c <= '9'

		break
	}
	if !firstDigit {
		return "", ErrInvalidPort
	}

	scheme := protocolValue
	if scheme == "" {
		// Use a non-special scheme so the URL parser does not treat a
		// well-known default port (http/80, https/443, ...) as empty.
		scheme = "urlpattern-non-special"
	}

	u, err := urlParser.Parse(scheme + "://dummy.test")
	if err != nil {
		return "", err
	}

	u, err = urlParser.BasicParser(portValue, nil, u, url.StatePort)
	if err != nil {
		return "", err
	}

	return u.Port(), nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-pathname
// TODO: Note, implementations are free to simply disable slash prepending in their URL parsing code instead of paying the performance penalty of inserting and removing characters in this algorithm.
func canonicalizePathname(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	leadingSlash := []rune(value)[0] == '/'
	var modifiedValue strings.Builder

	if !leadingSlash {
		modifiedValue.WriteString("/-")
	}

	modifiedValue.WriteString(value)

	dummyURL := urlParser.NewUrl()
	u, err := urlParser.BasicParser(modifiedValue.String(), nil, dummyURL, url.StatePathStart)
	if err != nil {
		return "", err
	}

	result := u.Pathname()

	if !leadingSlash {
		result = result[2:]
	}

	return result, nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-an-opaque-pathname
func canonicalizeOpaquePathname(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	var err error
	dummyURL := urlParser.NewUrl()

	u, err := urlParser.BasicParser(value, nil, dummyURL, url.StateOpaquePath)
	if err != nil {
		return "", err
	}

	return u.Pathname(), nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-search
func canonicalizeSearch(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	dummyURL := urlParser.NewUrl()

	u, err := urlParser.BasicParser(value, nil, dummyURL, url.StateQuery)
	if err != nil {
		return "", err
	}

	return u.Query(), nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-a-hash
func canonicalizeHash(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	dummyURL := urlParser.NewUrl()
	u, err := urlParser.BasicParser(value, nil, dummyURL, url.StateFragment)
	if err != nil {
		return "", err
	}

	return u.Fragment(), nil
}

// https://urlpattern.spec.whatwg.org/#canonicalize-an-ipv6-hostname
func canonicalizeIPv6Hostname(value string) (string, error) {
	var result strings.Builder

	for _, c := range value {
		if c != '[' && c != ']' && c != ':' && !unicode.Is(unicode.ASCII_Hex_Digit, c) {
			return "", ErrInvalidIPv6Hostname
		}

		result.WriteRune(unicode.ToLower(c))
	}

	return result.String(), nil
}
