/*
 * Copyright 2020 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package url

import (
	goerrors "errors"
	u2 "net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bits-and-blooms/bitset"

	"github.com/nlnwa/whatwg-url/errors"
)

func NewParser(opts ...ParserOption) Parser {
	p := &parser{opts: defaultParserOptions()}
	for _, opt := range opts {
		opt.apply(&p.opts)
	}
	return p
}

type Parser interface {
	Parse(rawUrl string) (*Url, error)
	ParseRef(rawUrl, ref string) (*Url, error)
	BasicParser(urlOrRef string, base *Url, url *Url, stateOverride State) (*Url, error)
	PercentEncodeString(s string, tr *PercentEncodeSet) string
	NewUrl() *Url
}

type parser struct {
	opts parserOptions
}

func (p *parser) Parse(rawUrl string) (*Url, error) {
	return p.BasicParser(rawUrl, nil, nil, NoState)
}

func (p *parser) ParseRef(rawUrl, ref string) (*Url, error) {
	if rawUrl == "" {
		return p.Parse(ref)
	}

	b, err := p.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	return p.BasicParser(ref, b, nil, NoState)
}

func (u *Url) Parse(ref string) (*Url, error) {
	return u.parser.BasicParser(ref, u, nil, NoState)
}

var defaultParser = NewParser()

func Parse(rawUrl string) (*Url, error) {
	return defaultParser.Parse(rawUrl)
}

func ParseRef(rawUrl, ref string) (*Url, error) {
	return defaultParser.ParseRef(rawUrl, ref)
}

type State int

const (
	NoState State = iota
	StateSchemeStart
	StateScheme
	StateNoScheme
	StateOpaquePath
	StateSpecialRelativeOrAuthority
	StateSpecialAuthoritySlashes
	StateSpecialAuthorityIgnoreSlashes
	StatePathOrAuthority
	StateAuthority
	StateHost
	StateHostname
	StateFile
	StateFileHost
	StateFileSlash
	StatePort
	StatePath
	StatePathStart
	StateQuery
	StateFragment
	StateRelative
	StateRelativeSlash
)

// BasicParser implements WHATWG basic URL parser (https://url.spec.whatwg.org/#concept-basic-url-parser)
// In most cases, when possible, prefer using the higher level Parse method.
func (p *parser) BasicParser(urlOrRef string, baseUrl *Url, url *Url, stateOverride State) (*Url, error) {
	stateOverridden := stateOverride > NoState
	if url == nil {
		url = &Url{inputUrl: urlOrRef, path: &path{}}
		if i, changed := trim(url.inputUrl, C0OrSpacePercentEncodeSet); changed {
			if err := p.handleError(url, errors.InvalidURLUnit, false); err != nil {
				return nil, err
			}
			url.inputUrl = i
		}
	} else {
		url.inputUrl = urlOrRef
	}
	url.parser = p

	if i, changed := remove(url.inputUrl, ASCIITabOrNewline); changed {
		if err := p.handleError(url, errors.InvalidURLUnit, false); err != nil {
			return nil, err
		}
		url.inputUrl = i
	}

	input := newInputString(url.inputUrl)
	var state State
	if stateOverridden {
		state = stateOverride
	} else {
		state = StateSchemeStart
	}

	var buffer strings.Builder
	atFlag := false
	bracketFlag := false
	passwordTokenSeenFlag := false

	var base *Url
	if baseUrl != nil {
		base = baseUrl.Clone()
	}

	for {
		r := input.nextCodePoint()

		switch state {
		case StateSchemeStart:
			if ASCIIAlpha.Test(uint(r)) {
				buffer.WriteRune(unicode.ToLower(r))
				state = StateScheme
			} else if !stateOverridden {
				state = StateNoScheme
				input.rewindLast()
			} else {
				if err := p.handleError(url, errors.InvalidURLUnit, true); err != nil {
					return nil, err
				}
			}
		case StateScheme:
			tr := ASCIIAlphanumeric.Clone().Set(0x2b).Set(0x2d).Set(0x2e)
			if tr.Test(uint(r)) {
				buffer.WriteRune(unicode.ToLower(r))
			} else if r == ':' {
				if stateOverridden {
					// If url’s scheme is a special scheme and buffer is not a special scheme, then return.
					if url.isSpecialScheme(url.scheme) && !url.isSpecialScheme(buffer.String()) {
						return url, nil
					}
					// If url’s scheme is not a special scheme and buffer is a special scheme, then return.
					if !url.isSpecialScheme(url.scheme) && url.isSpecialScheme(buffer.String()) {
						return url, nil
					}
					// If url includes credentials or has a non-null port, and buffer is "file", then return.
					if (url.username != "" || url.password != "" || url.port != nil) && buffer.String() == "file" {
						return url, nil
					}
					// If url’s scheme is "file" and its host is an empty host or null, then return.
					if url.scheme == "file" && *url.host == "" {
						return url, nil
					}
				}
				url.scheme = buffer.String()
				if stateOverridden {
					url.cleanDefaultPort()
					return url, nil
				}
				buffer.Reset()
				if url.scheme == "file" {
					if !input.remainingStartsWith("//") {
						if err := p.handleError(url, errors.SpecialSchemeMissingFollowingSolidus, false); err != nil {
							return nil, err
						}
					}
					state = StateFile
				} else if url.IsSpecialScheme() && base != nil && base.scheme == url.scheme {
					state = StateSpecialRelativeOrAuthority
				} else if url.IsSpecialScheme() {
					state = StateSpecialAuthoritySlashes
				} else if input.remainingStartsWith("/") {
					state = StatePathOrAuthority
					input.nextCodePoint()
				} else {
					url.path.setOpaque("")
					state = StateOpaquePath
				}
			} else if !stateOverridden {
				buffer.Reset()
				state = StateNoScheme
				input.reset()
			} else {
				if err := p.handleError(url, errors.InvalidURLUnit, true); err != nil {
					return nil, err
				}
			}
		case StateNoScheme:
			if base == nil || (base.path.isOpaque() && r != '#') {
				if err := p.handleError(url, errors.MissingSchemeNonRelativeURL, true); err != nil {
					return nil, err
				}
			} else if base.path.isOpaque() && r == '#' {
				url.scheme = base.scheme
				url.path = base.path
				url.query = base.query
				url.fragment = new(string)
				state = StateFragment
			} else if base.scheme != "file" {
				state = StateRelative
				input.rewindLast()
			} else {
				state = StateFile
				input.rewindLast()
			}
		case StateSpecialRelativeOrAuthority:
			if r == '/' && input.remainingStartsWith("/") {
				state = StateSpecialAuthorityIgnoreSlashes
				input.nextCodePoint()
			} else {
				if err := p.handleError(url, errors.SpecialSchemeMissingFollowingSolidus, false); err != nil {
					return nil, err
				}
				state = StateRelative
				input.rewindLast()
			}
		case StatePathOrAuthority:
			if r == '/' {
				state = StateAuthority
			} else {
				state = StatePath
				input.rewindLast()
			}
		case StateRelative:
			url.scheme = base.scheme
			if r == '/' {
				state = StateRelativeSlash
			} else if url.isSpecialSchemeAndBackslash(r) {
				if err := p.handleError(url, errors.InvalidReverseSolidus, false); err != nil {
					return nil, err
				}
				state = StateRelativeSlash
			} else {
				url.username = base.username
				url.password = base.password
				url.host = base.host
				url.port = base.port
				url.decodedPort = base.decodedPort
				url.path = base.path
				url.query = base.query
				if r == '?' {
					url.query = new(string)
					state = StateQuery
				} else if r == '#' {
					url.fragment = new(string)
					state = StateFragment
				} else if !input.eof {
					url.query = nil
					url.path.shortenPath(url.scheme)
					state = StatePath
					input.rewindLast()
				}
			}
		case StateRelativeSlash:
			if url.IsSpecialScheme() && (r == '/' || r == '\\') {
				if r == '\\' {
					if err := p.handleError(url, errors.InvalidReverseSolidus, false); err != nil {
						return nil, err
					}
				}
				state = StateSpecialAuthorityIgnoreSlashes
			} else if r == '/' {
				state = StateAuthority
			} else {
				url.username = base.username
				url.password = base.password
				url.host = base.host
				url.port = base.port
				url.decodedPort = base.decodedPort
				state = StatePath
				input.rewindLast()
			}
		case StateSpecialAuthoritySlashes:
			if r == '/' && input.remainingStartsWith("/") {
				state = StateSpecialAuthorityIgnoreSlashes
				input.nextCodePoint()
			} else {
				if err := p.handleError(url, errors.SpecialSchemeMissingFollowingSolidus, false); err != nil {
					return nil, err
				}
				state = StateSpecialAuthorityIgnoreSlashes
				input.rewindLast()
			}
		case StateSpecialAuthorityIgnoreSlashes:
			if r != '/' && r != '\\' {
				state = StateAuthority
				input.rewindLast()
			} else {
				if err := p.handleError(url, errors.SpecialSchemeMissingFollowingSolidus, false); err != nil {
					return nil, err
				}
			}
		case StateAuthority:
			if r == '@' {
				if err := p.handleError(url, errors.InvalidCredentials, false); err != nil {
					return nil, err
				}
				if atFlag {
					// Prepend %40 to buffer
					tmp := buffer.String()
					buffer.Reset()
					buffer.WriteString("%40")
					buffer.WriteString(tmp)
				}
				atFlag = true
				bb := newInputString(buffer.String())
				c := bb.nextCodePoint()
				for !bb.eof {
					if c == ':' && !passwordTokenSeenFlag {
						passwordTokenSeenFlag = true
						c = bb.nextCodePoint()
						continue
					}
					encodedCodePoints := p.percentEncodeRune(c, UserInfoPercentEncodeSet)
					if passwordTokenSeenFlag {
						url.password += encodedCodePoints
					} else {
						url.username += encodedCodePoints
					}
					c = bb.nextCodePoint()
				}
				buffer.Reset()
			} else if (input.eof || r == '/' || r == '?' || r == '#') || url.isSpecialSchemeAndBackslash(r) {
				if atFlag && buffer.Len() == 0 {
					if err := p.handleError(url, errors.InvalidCredentials, true); err != nil {
						return nil, err
					}
				}
				input.rewind(len([]rune(buffer.String())) + 1)
				buffer.Reset()
				state = StateHost
			} else {
				buffer.WriteRune(r)
			}
		case StateHost:
			fallthrough
		case StateHostname:
			if stateOverridden && url.scheme == "file" {
				input.rewindLast()
				state = StateFileHost
			} else if r == ':' && !bracketFlag {
				if buffer.Len() == 0 {
					if err := p.handleError(url, errors.HostMissing, true); err != nil {
						return nil, err
					}
				}
				if stateOverride == StateHostname {
					return url, nil
				}
				host, err := p.parseHost(url, p, buffer.String(), !url.IsSpecialScheme())
				if err != nil {
					return url, err
				}
				url.host = &host
				buffer.Reset()
				state = StatePort
			} else if input.eof || (r == '/' || r == '?' || r == '#' || url.isSpecialSchemeAndBackslash(r)) {
				input.rewindLast()
				if url.IsSpecialScheme() && buffer.Len() == 0 {
					if err := p.handleError(url, errors.HostMissing, true); err != nil {
						return nil, err
					}
				} else if stateOverridden && buffer.Len() == 0 && (url.username != "" || url.password != "" || url.port != nil) {
					return url, nil
				} else {
					host, err := p.parseHost(url, p, buffer.String(), !url.IsSpecialScheme())
					if err != nil {
						return url, err
					}
					url.host = &host
					buffer.Reset()
					state = StatePathStart
					if stateOverridden {
						return url, nil
					}
				}
			} else {
				if r == '[' {
					bracketFlag = true
				} else if r == ']' {
					bracketFlag = false
				}
				if input.currentIsInvalid() && p.opts.acceptInvalidCodepoints {
					buffer.WriteString(string([]byte{input.getCurrentAsByte()}))
				} else {
					buffer.WriteRune(r)
				}
			}
		case StatePort:
			if ASCIIDigit.Test(uint(r)) {
				buffer.WriteRune(r)
			} else if (input.eof || r == '/' || r == '?' || r == '#') || url.isSpecialSchemeAndBackslash(r) || stateOverridden {
				if buffer.Len() > 0 {
					port, err := strconv.Atoi(buffer.String())
					if port > 65535 || goerrors.Is(err, strconv.ErrRange) {
						if err := p.handleWrappedError(url, errors.PortOutOfRange, true, err); err != nil {
							return nil, err
						}
					}
					portString := strconv.Itoa(port)
					url.decodedPort = port
					url.port = &portString
					url.cleanDefaultPort()
					buffer.Reset()
				} else if stateOverridden {
					if err := p.handleError(url, errors.PortMissing, true); err != nil {
						return nil, err
					}
				}
				if stateOverridden {
					return url, nil
				}
				state = StatePathStart
				input.rewindLast()
			} else {
				if err := p.handleError(url, errors.PortInvalid, true); err != nil {
					return nil, err
				}
			}
		case StateFile:
			url.scheme = "file"
			url.host = new(string)
			if r == '/' || r == '\\' {
				if r == '\\' {
					if err := p.handleError(url, errors.InvalidReverseSolidus, false); err != nil {
						return nil, err
					}
				}
				state = StateFileSlash
			} else if base != nil && base.scheme == "file" {
				url.host = base.host
				url.path = base.path
				url.query = base.query
				if r == '?' {
					url.query = new(string)
					state = StateQuery
				} else if r == '#' {
					url.fragment = new(string)
					state = StateFragment
				} else if !input.eof {
					url.query = nil
					if !startsWithAWindowsDriveLetter(input.remainingFromPointer()) {
						url.path.shortenPath(url.scheme)
					} else {
						if err := p.handleError(url, errors.FileInvalidWindowsDriveLetter, false); err != nil {
							return nil, err
						}
						url.path.init()
					}
					state = StatePath
					input.rewindLast()
				}
			} else {
				state = StatePath
				input.rewindLast()
			}
		case StateFileSlash:
			if r == '/' || r == '\\' {
				if r == '\\' {
					if err := p.handleError(url, errors.InvalidReverseSolidus, false); err != nil {
						return nil, err
					}
				}
				state = StateFileHost
			} else {
				if base != nil && base.scheme == "file" {
					url.host = base.host
					if !startsWithAWindowsDriveLetter(input.remainingFromPointer()) && base.path != nil && isNormalizedWindowsDriveLetter(base.path.p[0]) {
						// This is a (platform-independent) Windows drive letter quirk. Both url’s and base’s host are null under these conditions and therefore not copied
						url.path.addSegment(base.path.p[0])
					}
				}
				state = StatePath
				input.rewindLast()
			}
		case StateFileHost:
			if input.eof || r == '/' || r == '\\' || r == '?' || r == '#' {
				input.rewindLast()
				if !stateOverridden && isWindowsDriveLetter(buffer.String()) {
					if err := p.handleError(url, errors.FileInvalidWindowsDriveLetterHost, false); err != nil {
						return nil, err
					}
					state = StatePath
				} else if buffer.Len() == 0 {
					url.host = new(string)
					if stateOverridden {
						return nil, nil
					}
					state = StatePathStart
				} else {
					host, err := p.parseHost(url, p, buffer.String(), !url.IsSpecialScheme())
					if err != nil {
						return url, err
					}
					if host == "localhost" {
						host = ""
					}
					url.host = &host
					if stateOverridden {
						return url, nil
					}
					buffer.Reset()
					state = StatePathStart
				}
			} else {
				buffer.WriteRune(r)
			}
		case StatePathStart:
			if url.IsSpecialScheme() && !p.opts.skipTrailingSlashNormalization {
				if r == '\\' {
					if err := p.handleError(url, errors.InvalidReverseSolidus, false); err != nil {
						return nil, err
					}
				}
				state = StatePath
				if r != '/' && r != '\\' {
					input.rewindLast()
				}
			} else if !stateOverridden && r == '?' {
				url.query = new(string)
				state = StateQuery
			} else if !stateOverridden && r == '#' {
				url.fragment = new(string)
				state = StateFragment
			} else if !input.eof {
				state = StatePath
				if r != '/' {
					input.rewindLast()
				}
			} else if stateOverridden && url.host == nil {
				url.path.addSegment("")
			}
		case StatePath:
			if (input.eof || r == '/') ||
				url.isSpecialSchemeAndBackslash(r) ||
				(!stateOverridden && (r == '?' || r == '#')) {

				if url.isSpecialSchemeAndBackslash(r) {
					if err := p.handleError(url, errors.InvalidReverseSolidus, false); err != nil {
						return nil, err
					}
				}
				if isDoubleDotPathSegment(buffer.String()) {
					url.path.shortenPath(url.scheme)

					if r != '/' && !url.isSpecialSchemeAndBackslash(r) {
						url.path.addSegment("")
					}
				} else if isSingleDotPathSegment(buffer.String()) && r != '/' && !url.isSpecialSchemeAndBackslash(r) {
					url.path.addSegment("")
				} else if !isSingleDotPathSegment(buffer.String()) {
					if url.scheme == "file" && url.path.isEmpty() && isWindowsDriveLetter(buffer.String()) {
						// replace second code point in buffer with U+003A (:).
						// This is a (platform-independent) Windows drive letter quirk.
						if !p.opts.skipWindowsDriveLetterNormalization {
							b := buffer.String()
							buffer.Reset()
							buffer.WriteString(b[0:1] + ":" + b[2:])
						}
					}
					if !p.opts.collapseConsecutiveSlashes || !url.IsSpecialScheme() || url.path.isEmpty() || len(url.path.p[len(url.path.p)-1]) > 0 {
						url.path.addSegment(buffer.String())
					} else {
						url.path.p[len(url.path.p)-1] = buffer.String()
					}
				}
				buffer.Reset()
				if r == '?' {
					url.query = new(string)
					state = StateQuery
				} else if r == '#' {
					url.fragment = new(string)
					state = StateFragment
				}
			} else {
				if !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.InvalidURLUnit, false); err != nil {
						return nil, err
					}
				}
				invalidPercentEncoding, d := input.remainingIsInvalidPercentEncoded()
				if invalidPercentEncoding {
					if err := p.handleErrorWithDescription(url, errors.InvalidURLUnit, false, d); err != nil {
						return nil, err
					}
				}
				if invalidPercentEncoding {
					buffer.WriteString(p.percentEncodeInvalidRune(r, p.opts.pathPercentEncodeSet))
				} else {
					buffer.WriteString(p.percentEncodeRune(r, p.opts.pathPercentEncodeSet))
				}
			}
		case StateOpaquePath:
			if r == '?' {
				url.query = new(string)
				state = StateQuery
				buffer.Reset()
			} else if r == '#' {
				url.fragment = new(string)
				state = StateFragment
				buffer.Reset()
			} else if !input.eof {
				if !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.InvalidURLUnit, false); err != nil {
						return nil, err
					}
				}
				invalidPercentEncoding, d := input.remainingIsInvalidPercentEncoded()
				if invalidPercentEncoding {
					if err := p.handleErrorWithDescription(url, errors.InvalidURLUnit, false, d); err != nil {
						return nil, err
					}
					buffer.WriteString(p.percentEncodeInvalidRune(r, C0PercentEncodeSet))
				} else {
					buffer.WriteString(p.percentEncodeRune(r, C0PercentEncodeSet))
				}
				url.path.setOpaque(buffer.String())
			}
		case StateQuery:
			if !stateOverridden && r == '#' {
				url.fragment = new(string)
				state = StateFragment
				*url.query = buffer.String()
				buffer.Reset()
			} else if !input.eof {
				if !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.InvalidURLUnit, false); err != nil {
						return nil, err
					}
				}
				invalidPercentEncoding, d := input.remainingIsInvalidPercentEncoded()
				if invalidPercentEncoding {
					if err := p.handleErrorWithDescription(url, errors.InvalidURLUnit, false, d); err != nil {
						return nil, err
					}
				}
				encodeSet := p.opts.queryPercentEncodeSet
				if url.isSpecialScheme(url.scheme) {
					encodeSet = p.opts.specialQueryPercentEncodeSet
				}
				buffer.WriteString(p.percentEncodeRune(r, encodeSet))
			} else {
				q := buffer.String()
				url.query = &q
			}
		case StateFragment:
			if !input.eof {
				if !isURLCodePoint(r) && r != '%' {
					if err := p.handleError(url, errors.InvalidURLUnit, false); err != nil {
						return nil, err
					}
				}
				invalidPercentEncoding, d := input.remainingIsInvalidPercentEncoded()
				if invalidPercentEncoding {
					if err := p.handleErrorWithDescription(url, errors.InvalidURLUnit, false, d); err != nil {
						return nil, err
					}
				}
				encodeSet := p.opts.fragmentPercentEncodeSet
				if url.isSpecialScheme(url.scheme) {
					encodeSet = p.opts.specialFragmentPercentEncodeSet
				}
				buffer.WriteString(p.percentEncodeRune(r, encodeSet))
			} else {
				f := buffer.String()
				url.fragment = &f
			}
		}

		if input.eof {
			break
		}
	}

	return url, nil
}

func (p *parser) percentEncodeInvalidRune(r rune, tr *PercentEncodeSet) string {
	if p.opts.percentEncodeSinglePercentSign {
		return p.percentEncodeRune(r, tr.Set(0x25))
	}
	return p.percentEncodeRune(r, tr)
}

func (p *parser) percentEncodeRune(r rune, tr *PercentEncodeSet) string {
	if tr != nil && !tr.RuneShouldBeEncoded(r) {
		return string(r)
	}

	var bytes = make([]byte, 4)
	var n int
	if p.opts.encodingOverride != nil {
		b, _ := p.opts.encodingOverride.EncodeRune(r)
		bytes[0] = b
		n = 1
	} else {
		n = utf8.EncodeRune(bytes, r)
	}

	percentEncoded := make([]byte, 4*3)
	j := 0
	for i := 0; i < n; i++ {
		c := bytes[i]
		percentEncoded[j] = '%'
		percentEncoded[j+1] = "0123456789ABCDEF"[c>>4]
		percentEncoded[j+2] = "0123456789ABCDEF"[c&15]
		j += 3
	}
	return string(percentEncoded[:j])
}

func (p *parser) PercentEncodeString(s string, tr *PercentEncodeSet) string {
	buffer := &strings.Builder{}
	runes := []rune(s)
	for i, r := range runes {
		if r == '%' {
			if len(runes) < (i+3) ||
				(!ASCIIHexDigit.Test(uint(runes[i+1])) || !ASCIIHexDigit.Test(uint(runes[i+2]))) {
				if p.opts.percentEncodeSinglePercentSign {
					buffer.WriteString(p.percentEncodeRune(r, tr.Set(0x25)))
					continue
				}
			}
		}
		buffer.WriteString(p.percentEncodeRune(r, tr))
	}
	return buffer.String()
}

func (p *parser) DecodePercentEncoded(s string) string {
	sb := strings.Builder{}
	bytes := []byte(s)
	for i := 0; i < len(bytes); i++ {
		if bytes[i] != '%' {
			sb.WriteByte(bytes[i])
		} else if len(bytes) < (i+3) ||
			(!ASCIIHexDigit.Test(uint(bytes[i+1])) || !ASCIIHexDigit.Test(uint(bytes[i+2]))) {
			sb.WriteByte(bytes[i])
		} else {
			b, e := u2.PathUnescape(string(bytes[i : i+3]))
			if e != nil {
				return sb.String()
			}
			if p.opts.encodingOverride != nil {
				r := p.opts.encodingOverride.DecodeByte(b[0])
				sb.WriteRune(r)
			} else {
				sb.WriteString(b)
			}
			i += 2
		}
	}
	return sb.String()
}

func (p *parser) NewUrl() *Url {
	u := Url{}
	u.parser = p

	u.path = &path{}
	u.path.init()

	return &u
}

func isSingleDotPathSegment(s string) bool {
	if s == "." {
		return true
	}
	s = strings.ToLower(s)
	return s == "%2e"
}

func isDoubleDotPathSegment(s string) bool {
	if s == ".." {
		return true
	}
	s = strings.ToLower(s)
	if s == ".%2e" || s == "%2e." || s == "%2e%2e" {
		return true
	}
	return false
}

func startsWithAWindowsDriveLetter(s string) bool {
	if len(s) >= 2 && isWindowsDriveLetter(s[0:2]) &&
		(len(s) == 2 || s[2] == '/' || s[2] == '\\' || s[2] == '?' || s[2] == '#') {
		return true
	}

	return false
}

func isWindowsDriveLetter(s string) bool {
	if len(s) == 2 && ASCIIAlpha.Test(uint(s[0])) &&
		(s[1] == ':' || s[1] == '|') {
		return true
	}
	return false
}

func isNormalizedWindowsDriveLetter(s string) bool {
	if len(s) == 2 && ASCIIAlpha.Test(uint(s[0])) &&
		(s[1] == ':') {
		return true
	}
	return false
}

func trimPrefix(s string, tr *PercentEncodeSet) (string, bool) {
	if s == "" {
		return s, false
	}
	for i, c := range s {
		if tr.RuneNotInSet(c) {
			return s[i:], i > 0
		}
	}
	return "", true
}

func trimPostfix(s string, tr *PercentEncodeSet) (string, bool) {
	if s == "" {
		return s, false
	}
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if tr.RuneNotInSet(int32(c)) {
			return s[:i+1], i < (len(s) - 1)
		}
	}
	return "", true
}

func trim(s string, tr *PercentEncodeSet) (string, bool) {
	var c1, c2 bool
	s, c1 = trimPrefix(s, tr)
	s, c2 = trimPostfix(s, tr)
	return s, c1 || c2
}

func remove(s string, tr *bitset.BitSet) (string, bool) {
	if s == "" {
		return s, false
	}
	changed := false
	var r []byte
	for _, c := range []byte(s) {
		if tr.Test(uint(c)) {
			changed = true
		} else {
			r = append(r, c)
		}
	}
	return string(r), changed
}

func containsOnly(s string, tr *bitset.BitSet) bool {
	for _, c := range []byte(s) {
		if !tr.Test(uint(c)) {
			return false
		}
	}
	return true
}

func (u *Url) IsSpecialScheme() bool {
	return u.isSpecialScheme(u.scheme)
}

func (u *Url) isSpecialScheme(s string) bool {
	_, ok := u.getSpecialScheme(s)
	return ok
}

func (u *Url) getSpecialScheme(s string) (string, bool) {
	dp, ok := u.parser.opts.specialSchemes[s]
	return dp, ok
}

func (u *Url) isSpecialSchemeAndBackslash(r rune) bool {
	ok := u.IsSpecialScheme()
	return ok && r == '\\'
}

func (u *Url) cleanDefaultPort() {
	if dp, ok := u.getSpecialScheme(u.scheme); ok && (u.port == nil || dp == *u.port) {
		u.port = nil
		u.decodedPort = 0
	}
}

func (u *Url) getDefaultPort() int {
	if dp, ok := u.getSpecialScheme(u.scheme); ok {
		if p, err := strconv.Atoi(dp); err == nil {
			return p
		}
	}
	return 0
}
