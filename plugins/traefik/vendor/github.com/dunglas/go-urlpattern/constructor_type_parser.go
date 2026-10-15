package urlpattern

import (
	"regexp"

	"golang.org/x/exp/utf8string"
)

type state uint8

// https://urlpattern.spec.whatwg.org/#constructor-string-parsing

type constructorTypeParser struct {
	input                         utf8string.String
	tokenList                     []token
	result                        URLPatternInit
	componentStart                int
	tokenIndex                    int
	tokenIncrement                int
	groupDepth                    int
	hostnameIPv6BracketDepth      int
	protocolMatchesASpecialScheme bool
	state                         state
}

// https://urlpattern.spec.whatwg.org/#constructor-string-parser-state

const (
	stateInit state = iota
	stateProtocol
	sateAuthority
	stateUsername
	statePassword
	stateHostname
	statePort
	statePathname
	stateSearch
	stateHash
	stateDone
)

// https://urlpattern.spec.whatwg.org/#parse-a-constructor-string
func newConstructorTypeParser(input string, tokenList []token) constructorTypeParser {
	return constructorTypeParser{
		input:          *utf8string.NewString(input),
		tokenList:      tokenList,
		result:         URLPatternInit{},
		tokenIncrement: 1,
		state:          stateInit,
	}
}

// https://urlpattern.spec.whatwg.org/#constructor-string-parsing
func parseConstructorString(input string) (*URLPatternInit, error) {
	tl, err := tokenize(input, tokenizePolicyLenient)
	if err != nil {
		return nil, err
	}

	p := newConstructorTypeParser(input, tl)

	tlLen := len(p.tokenList)

	for p.tokenIndex < tlLen {
		p.tokenIncrement = 1

		if p.tokenList[p.tokenIndex].tType == tokenEnd {
			if p.state == stateInit {
				p.rewind()

				switch {
				case p.isHashPrefix():
					p.changeState(stateHash, 1)
				case p.isSearchPrefix():
					p.changeState(stateSearch, 1)
				default:
					p.changeState(statePathname, 0)
				}

				p.tokenIndex += p.tokenIncrement

				continue
			}

			if p.state == sateAuthority {
				p.rewindAndSetState(stateHostname)
				p.tokenIndex += p.tokenIncrement

				continue
			}

			p.changeState(stateDone, 0)

			break
		}

		if p.isGroupOpen() {
			p.groupDepth++
			p.tokenIndex += p.tokenIncrement

			continue
		}

		if p.groupDepth > 0 {
			if p.isGroupClose() {
				p.groupDepth--
			} else {
				p.tokenIndex += p.tokenIncrement

				continue
			}
		}

		// Switch on parser’s state and run the associated steps:
		switch p.state {
		case stateInit:
			if p.isProtocolSuffix() {
				p.rewindAndSetState(stateProtocol)
			}
		case stateProtocol:
			if p.isProtocolSuffix() {
				if err := p.computeProtocolMatchesSpecialSchemeFlag(); err != nil {
					return nil, err
				}

				nextState := statePathname
				skip := 1

				if p.nextIsAuthoritySlashes() {
					nextState = sateAuthority
					skip = 3
				} else if p.protocolMatchesASpecialScheme {
					nextState = sateAuthority
				}

				p.changeState(nextState, skip)
			}

		case sateAuthority:
			if p.isIdentityTerminator() {
				p.rewindAndSetState(stateUsername)
			} else if p.isPathnameStart() ||
				p.isSearchPrefix() ||
				p.isHashPrefix() {
				p.rewindAndSetState(stateHostname)
			}

		case stateUsername:
			if p.isPasswordPrefix() {
				p.changeState(statePassword, 1)
			} else if p.isIdentityTerminator() {
				p.changeState(stateHostname, 1)
			}

		case statePassword:
			if p.isIdentityTerminator() {
				p.changeState(stateHostname, 1)
			}

		case stateHostname:
			switch {
			case p.isIPV6Open():
				p.hostnameIPv6BracketDepth++
			case p.isIPV6Close():
				p.hostnameIPv6BracketDepth--
			case p.isPortPrefix() && p.hostnameIPv6BracketDepth == 0:
				p.changeState(statePort, 1)
			case p.isPathnameStart():
				p.changeState(statePathname, 0)
			case p.isSearchPrefix():
				p.changeState(stateSearch, 1)
			case p.isHashPrefix():
				p.changeState(stateHash, 1)
			}

		case statePort:
			switch {
			case p.isPathnameStart():
				p.changeState(statePathname, 0)
			case p.isSearchPrefix():
				p.changeState(stateSearch, 1)
			case p.isHashPrefix():
				p.changeState(stateHash, 1)
			}

		case statePathname:
			if p.isSearchPrefix() {
				p.changeState(stateSearch, 1)
			} else if p.isHashPrefix() {
				p.changeState(stateHash, 1)
			}

		case stateSearch:
			if p.isHashPrefix() {
				p.changeState(stateHash, 1)
			}

		case stateHash:
			// noop

		case stateDone:
			// Assert: This step is never reached.
			panic("done state must never be reached")
		}

		p.tokenIndex += p.tokenIncrement
	}

	// If parser’s result contains "hostname" and not "port", then set parser’s result["port"] to the empty string.
	if p.result.Hostname != nil && p.result.Port == nil {
		es := ""
		p.result.Port = &es
	}

	return &p.result, nil
}

// https://urlpattern.spec.whatwg.org/#rewind
func (p *constructorTypeParser) rewind() {
	p.tokenIndex = p.componentStart
	p.tokenIncrement = 0
}

// https://urlpattern.spec.whatwg.org/#rewind-and-set-state
func (p *constructorTypeParser) rewindAndSetState(s state) {
	p.rewind()
	p.state = s
}

// https://urlpattern.spec.whatwg.org/#is-a-hash-prefix
func (p *constructorTypeParser) isHashPrefix() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, "#")
}

// https://urlpattern.spec.whatwg.org/#is-a-search-prefix
func (p *constructorTypeParser) isSearchPrefix() bool {
	if p.isNonSpecialPatternChar(p.tokenIndex, "?") {
		return true
	}

	if p.tokenList[p.tokenIndex].value != "?" {
		return false
	}

	previousIndex := p.tokenIndex - 1
	if previousIndex < 0 {
		return true
	}

	previousToken := p.getSafeToken(previousIndex)
	switch previousToken.tType {
	case tokenName:
		return false

	case tokenRegexp:
		return false

	case tokenClose:
		return false

	case tokenAsterisk:
		return false
	}

	return true
}

// https://urlpattern.spec.whatwg.org/#is-a-group-open
func (p *constructorTypeParser) isGroupOpen() bool {
	return p.tokenList[p.tokenIndex].tType == tokenOpen
}

// https://urlpattern.spec.whatwg.org/#is-a-group-close
func (p *constructorTypeParser) isGroupClose() bool {
	return p.tokenList[p.tokenIndex].tType == tokenClose
}

// https://urlpattern.spec.whatwg.org/#is-a-non-special-pattern-char
func (p *constructorTypeParser) isNonSpecialPatternChar(index int, value string) bool {
	token := p.getSafeToken(index)
	if token.value != value {
		return false
	}

	return token.tType == tokenChar || token.tType == tokenEscapedChar || token.tType == tokenInvalidChar
}

// https://urlpattern.spec.whatwg.org/#get-a-safe-token
func (p *constructorTypeParser) getSafeToken(index int) token {
	len := len(p.tokenList)

	if index < len {
		return p.tokenList[index]
	}

	// Assert: parser's token list's size is greater than or equal to 1.

	return p.tokenList[len-1]
}

// https://urlpattern.spec.whatwg.org/#change-state
func (p *constructorTypeParser) changeState(newState state, skip int) {
	v := p.makeComponentString()

	// ignore sInit, authority and done
	switch p.state {
	case stateProtocol:
		p.result.Protocol = &v
	case stateUsername:
		p.result.Username = &v
	case statePassword:
		p.result.Password = &v
	case stateHostname:
		p.result.Hostname = &v
	case statePort:
		p.result.Port = &v
	case statePathname:
		p.result.Pathname = &v
	case stateSearch:
		p.result.Search = &v
	case stateHash:
		p.result.Hash = &v
	}

	if p.state != stateInit && newState != stateDone {
		es := ""

		// If parser’s state is "protocol", "authority", "username", or "password"; new state is "port", "pathname", "search", or "hash"; and parser’s result["hostname"] does not exist, then set parser’s result["hostname"] to the empty string.
		if p.result.Hostname == nil &&
			(p.state == stateProtocol || p.state == sateAuthority || p.state == stateUsername || p.state == statePassword) &&
			(newState == statePort || newState == statePathname || newState == stateSearch || newState == stateHash) {
			p.result.Hostname = &es
		}

		if p.result.Pathname == nil &&
			(p.state == stateProtocol || p.state == sateAuthority || p.state == stateUsername || p.state == statePassword || p.state == stateHostname || p.state == statePort) &&
			(newState == stateSearch || newState == stateHash) {
			if p.protocolMatchesASpecialScheme {
				sl := "/"
				p.result.Pathname = &sl
			} else {
				p.result.Pathname = &es
			}
		}

		if p.result.Search == nil &&
			(p.state == stateProtocol || p.state == sateAuthority || p.state == stateUsername || p.state == statePassword || p.state == stateHostname || p.state == statePort || p.state == statePathname) &&
			(newState == stateHash) {
			p.result.Search = &es
		}
	}

	p.state = newState
	p.tokenIndex += skip
	p.componentStart = p.tokenIndex
	p.tokenIncrement = 0
}

// https://urlpattern.spec.whatwg.org/#make-a-component-string
func (p *constructorTypeParser) makeComponentString() string {
	token := p.tokenList[p.tokenIndex]
	componentStartToken := p.getSafeToken(p.componentStart)
	componentStartInputIndex := componentStartToken.index
	endIndex := token.index

	return p.input.Slice(componentStartInputIndex, endIndex)
}

// https://urlpattern.spec.whatwg.org/#is-a-protocol-suffix
func (p *constructorTypeParser) isProtocolSuffix() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, ":")
}

// https://urlpattern.spec.whatwg.org/#compute-protocol-matches-a-special-scheme-flag
func (p *constructorTypeParser) computeProtocolMatchesSpecialSchemeFlag() error {
	protocol := p.makeComponentString()
	protocolComponent, err := compileComponent(protocol, canonicalizeProtocol, options{})
	if err != nil {
		return err
	}

	if protocolComponent.protocolComponentMatchesSpecialScheme() {
		p.protocolMatchesASpecialScheme = true
	}

	return nil
}

// https://urlpattern.spec.whatwg.org/#next-is-authority-slashes
func (p *constructorTypeParser) nextIsAuthoritySlashes() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex+1, "/") && p.isNonSpecialPatternChar(p.tokenIndex+2, "/")
}

// https://urlpattern.spec.whatwg.org/#is-an-identity-terminator
func (p *constructorTypeParser) isIdentityTerminator() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, "@")
}

// https://urlpattern.spec.whatwg.org/#is-a-pathname-start
func (p *constructorTypeParser) isPathnameStart() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, "/")
}

// https://urlpattern.spec.whatwg.org/#is-a-password-prefix
func (p *constructorTypeParser) isPasswordPrefix() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, ":")
}

// https://urlpattern.spec.whatwg.org/#is-a-port-prefix
func (p *constructorTypeParser) isPortPrefix() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, ":")
}

// https://urlpattern.spec.whatwg.org/#is-an-ipv6-open
func (p *constructorTypeParser) isIPV6Open() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, "[")
}

// https://urlpattern.spec.whatwg.org/#is-an-ipv6-close
func (p *constructorTypeParser) isIPV6Close() bool {
	return p.isNonSpecialPatternChar(p.tokenIndex, "]")
}

// Precompiled regular expressions for the component patterns that recur across
// nearly every URLPattern: the full wildcard (the default for username,
// password, search and hash) and the empty component (a common port). Compiled
// once at init; a *regexp.Regexp is safe for concurrent use, so the same
// instance is shared by every component whose generated expression matches.
var sharedComponentRegexps = map[string]*regexp.Regexp{ //nolint:gochecknoglobals
	`\A(?:(.*))\z`:     regexp.MustCompile(`\A(?:(.*))\z`),
	`(?i)\A(?:(.*))\z`: regexp.MustCompile(`(?i)\A(?:(.*))\z`),
	`\A(?:)\z`:         regexp.MustCompile(`\A(?:)\z`),
	`(?i)\A(?:)\z`:     regexp.MustCompile(`(?i)\A(?:)\z`),
}

// https://urlpattern.spec.whatwg.org/#compile-a-component
func compileComponent(input string, encodencodingCallback encodingCallback, options options) (*component, error) {
	partList, err := parsePatternString(input, options, encodencodingCallback)
	if err != nil {
		return nil, err
	}

	// Let (regular expression string, name list) be the result of running generate a regular expression and name list given part list and options.
	regularExpressionString, nameList, err := partList.generateRegularExpressionAndNameList(options)
	if err != nil {
		return nil, err
	}

	// Most components of a typical pattern are the full wildcard or empty:
	// username, password, search and hash default to the full wildcard and the
	// port is often empty. Their generated regular expressions are a tiny fixed
	// set of constant strings, so reuse a shared precompiled *regexp.Regexp
	// (safe for concurrent use) instead of paying regexp.Compile per component.
	regularExpression, ok := sharedComponentRegexps[regularExpressionString]
	if !ok {
		var err error

		regularExpression, err = regexp.Compile(regularExpressionString)
		if err != nil {
			return nil, err
		}
	}

	patternString, err := partList.generatePatternString(options)
	if err != nil {
		return nil, err
	}

	var hasRegexpGroups bool
	for _, part := range partList {
		if part.pType == partRegexp {
			hasRegexpGroups = true

			break
		}
	}

	return &component{patternString, regularExpression, nameList, hasRegexpGroups}, nil
}
