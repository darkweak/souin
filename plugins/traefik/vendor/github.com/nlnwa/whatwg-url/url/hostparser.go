/*
 * Copyright 2019 National Library of Norway.
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
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/idna"

	"github.com/nlnwa/whatwg-url/errors"
)

// parseHost parses the host part of the input string.
func (p *parser) parseHost(u *Url, parser *parser, input string, isNotSpecial bool) (string, error) {
	if p.opts.preParseHostFunc != nil {
		input = p.opts.preParseHostFunc(u, input)
	}
	if input == "" {
		return "", nil
	}
	if input[0] == '[' {
		if !strings.HasSuffix(input, "]") {
			if err := p.handleError(u, errors.IPv6Unclosed, true); err != nil {
				return "", err
			}
		}
		input = strings.Trim(input, "[]")
		return p.parseIPv6(u, newInputString(input))
	}
	if isNotSpecial {
		return p.parseOpaqueHost(u, input)
	}

	domain := p.DecodePercentEncoded(input)

	if !utf8.ValidString(domain) {
		if p.opts.laxHostParsing {
			return percentEncodeString(input, HostPercentEncodeSet), nil
		}
		if err := p.handleErrorWithDescription(u, errors.DomainToASCII, true, "not a valid UTF-8 string"); err != nil {
			return "", err
		}
	}

	asciiDomain, err := p.ToASCII(domain, false)
	if err != nil {
		if p.opts.laxHostParsing {
			return domain, nil
		}
		if err := p.handleWrappedError(u, errors.DomainToASCII, true, err); err != nil {
			return "", err
		}
	}
	for _, c := range asciiDomain {
		if ForbiddenDomainCodePoint.Test(uint(c)) {
			if p.opts.laxHostParsing {
				return parser.PercentEncodeString(asciiDomain, HostPercentEncodeSet), nil
			} else {
				if err := p.handleErrorWithDescription(u, errors.DomainInvalidCodePoint, true, string(c)); err != nil {
					return "", err
				}
			}
		}
	}

	if p.endsInANumber(u, asciiDomain) {
		ipv4Host, err := p.parseIPv4(u, asciiDomain)
		return ipv4Host, err
	}

	if p.opts.postParseHostFunc != nil {
		asciiDomain = p.opts.postParseHostFunc(u, asciiDomain)
	}
	return asciiDomain, nil
}

func (p *parser) endsInANumber(u *Url, input string) bool {
	parts := strings.Split(input, ".")
	if parts[len(parts)-1] == "" {
		if len(parts) == 1 {
			return false
		}
		parts = parts[0 : len(parts)-1]
	}
	last := parts[len(parts)-1]
	if last != "" && containsOnly(last, ASCIIDigit) {
		return true
	}
	if _, _, err := p.parseIPv4Number(u, last); err == nil || goerrors.Is(err, strconv.ErrRange) {
		return true
	}
	return false
}

func (p *parser) parseIPv4Number(u *Url, input string) (number int64, validationError bool, err error) {
	if input == "" {
		if err = p.handleError(u, errors.IPv4EmptyPart, true); err != nil {
			return
		}
	}
	R := 10
	if len(input) >= 2 && (strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X")) {
		validationError = true
		input = input[2:]
		R = 16
	} else if len(input) >= 2 && strings.HasPrefix(input, "0") {
		validationError = true
		input = input[1:]
		R = 8
	}
	if input == "" {
		validationError = true
		return
	}
	number, err = strconv.ParseInt(input, R, 64)
	return
}

func (p *parser) parseIPv4(u *Url, input string) (string, error) {
	parts := strings.Split(input, ".")
	if parts[len(parts)-1] == "" {
		if err := p.handleError(u, errors.IPv4EmptyPart, false); err != nil {
			return input, err
		}
		if len(parts) > 1 {
			parts = parts[:len(parts)-1]
		}
	}
	if len(parts) > 4 {
		if err := p.handleError(u, errors.IPv4TooManyParts, true); err != nil {
			return input, err
		}
	}
	var numbers []int64
	for _, part := range parts {
		n, validationError, err := p.parseIPv4Number(u, part)
		if err != nil {
			if err := p.handleWrappedError(u, errors.IPv4NonNumericPart, true, err); err != nil {
				return input, err
			}
		}
		if validationError {
			if err := p.handleError(u, errors.IPv4NonDecimalPart, false); err != nil {
				return input, err
			}
		}
		numbers = append(numbers, n)
	}
	for _, n := range numbers {
		if n > 255 {
			if err := p.handleError(u, errors.IPv4OutOfRangePart, false); err != nil {
				return input, err
			}
		}
	}
	for _, n := range numbers[:len(numbers)-1] {
		if n > 255 {
			if err := p.handleError(u, errors.IPv4OutOfRangePart, true); err != nil {
				return "", err
			}
		}
	}
	if numbers[len(numbers)-1] >= int64(math.Pow(256, float64(5-len(numbers)))) {
		if err := p.handleError(u, errors.IPv4OutOfRangePart, true); err != nil {
			return "", err
		}
	}
	var ipv4 = IPv4Addr(numbers[len(numbers)-1])
	numbers = numbers[:len(numbers)-1]
	for counter, n := range numbers {
		ipv4 += IPv4Addr(n * int64(math.Pow(256, float64(3-counter))))
	}
	u.isIPv4 = true
	return ipv4.String(), nil
}

func (p *parser) parseIPv6(u *Url, input *inputString) (string, error) {
	address := &IPv6Addr{}
	pieceIdx := 0
	compress := -1

	c := input.nextCodePoint()
	if c == ':' {
		if !input.remainingStartsWith(":") {
			if err := p.handleError(u, errors.IPv6InvalidCompression, true); err != nil {
				return "", err
			}
		}
		input.nextCodePoint()
		c = input.nextCodePoint()
		pieceIdx++
		compress = pieceIdx
	}
	for !input.eof {
		if pieceIdx == 8 {
			if err := p.handleError(u, errors.IPv6TooManyPieces, true); err != nil {
				return "", err
			}
		}
		if c == ':' {
			if compress >= 0 {
				if err := p.handleError(u, errors.IPv6MultipleCompression, true); err != nil {
					return "", err
				}
			}
			c = input.nextCodePoint()
			pieceIdx++
			compress = pieceIdx
			continue
		}
		value := 0
		length := 0

		for length < 4 && ASCIIHexDigit.Test(uint(c)) {
			v, _ := strconv.ParseInt(string(c), 16, 32)
			value = value*0x10 + int(v)
			c = input.nextCodePoint()
			length++
		}

		if c == '.' {
			if length == 0 {
				if err := p.handleError(u, errors.IPv4InIPv6InvalidCodePoint, true); err != nil {
					return "", err
				}
			}
			input.rewind(length + 1)
			c = input.nextCodePoint()
			if pieceIdx > 6 {
				if err := p.handleError(u, errors.IPv4InIPv6TooManyPieces, true); err != nil {
					return "", err
				}
			}
			numbersSeen := 0
			for !input.eof {
				ipv4Piece := -1
				if numbersSeen > 0 {
					if c == '.' && numbersSeen < 4 {
						c = input.nextCodePoint()
					} else {
						if err := p.handleError(u, errors.IPv4InIPv6InvalidCodePoint, true); err != nil {
							return "", err
						}
					}
				}
				if !ASCIIDigit.Test(uint(c)) {
					if err := p.handleError(u, errors.IPv4InIPv6InvalidCodePoint, true); err != nil {
						return "", err
					}
				}
				for ASCIIDigit.Test(uint(c)) {
					number, _ := strconv.Atoi(string(c))
					if ipv4Piece < 0 {
						ipv4Piece = number
					} else if ipv4Piece == 0 {
						if err := p.handleError(u, errors.IPv4InIPv6InvalidCodePoint, true); err != nil {
							return "", err
						}
					} else {
						ipv4Piece = ipv4Piece*10 + number
					}

					if ipv4Piece > 255 {
						if err := p.handleError(u, errors.IPv4InIPv6OutOfRangePart, true); err != nil {
							return "", err
						}
					}
					c = input.nextCodePoint()
				}
				address[pieceIdx] = address[pieceIdx]*0x100 + uint16(ipv4Piece)
				numbersSeen++
				if numbersSeen == 2 || numbersSeen == 4 {
					pieceIdx++
				}
			}
			if numbersSeen != 4 {
				if err := p.handleError(u, errors.IPv4InIPv6TooFewParts, true); err != nil {
					return "", err
				}
			}
			break
		} else if c == ':' {
			c = input.nextCodePoint()
			if input.eof {
				if err := p.handleError(u, errors.IPv6InvalidCodePoint, true); err != nil {
					return "", err
				}
			}
		} else if !input.eof {
			if err := p.handleError(u, errors.IPv6InvalidCodePoint, true); err != nil {
				return "", err
			}
		}
		address[pieceIdx] = uint16(value)
		pieceIdx++
	}
	if compress >= 0 {
		swaps := pieceIdx - compress
		pieceIdx = 7
		for pieceIdx != 0 && swaps > 0 {
			t := address[pieceIdx]
			address[pieceIdx] = address[compress+swaps-1]
			address[compress+swaps-1] = t
			pieceIdx--
			swaps--
		}
	} else if compress < 0 && pieceIdx != 8 {
		if err := p.handleError(u, errors.IPv6TooFewPieces, true); err != nil {
			return "", err
		}
	}
	u.isIPv6 = true
	return "[" + address.String() + "]", nil
}

func (p *parser) parseOpaqueHost(u *Url, input string) (string, error) {
	output := ""
	for i, c := range input {
		if ForbiddenHostCodePoint.Test(uint(c)) {
			if p.opts.laxHostParsing {
				return input, nil
			} else {
				if err := p.handleErrorWithDescription(u, errors.HostInvalidCodePoint, true, string(c)); err != nil {
					return "", err
				}
			}
		}
		if !isURLCodePoint(c) && c != '%' {
			if err := p.handleErrorWithDescription(u, errors.InvalidURLUnit, false, string(c)); err != nil {
				return "", err
			}
		}
		if c == '%' {
			invalidPercentEncoding, d := remainingIsInvalidPercentEncoded([]rune(input[i:]))
			if invalidPercentEncoding {
				if err := p.handleErrorWithDescription(u, errors.InvalidURLUnit, false, d); err != nil {
					return "", err
				}
			}
		}

		output += p.percentEncodeRune(c, C0PercentEncodeSet)
	}
	return output, nil
}

type IPv6Addr [8]uint16

func (address *IPv6Addr) String() string {
	output := ""
	compress := -1

	currentIdx := -1
	currentLength := 0
	compressLength := 0
	for pieceIdx := 0; pieceIdx < 8; pieceIdx++ {
		if address[pieceIdx] == 0 {
			if currentIdx < 0 {
				currentIdx = pieceIdx
			}
			currentLength++
		} else {
			if currentLength > 1 && currentLength > compressLength {
				compress = currentIdx
				compressLength = currentLength
			}
			currentIdx = -1
			currentLength = 0
		}
	}
	if currentLength > 1 && currentLength > compressLength {
		compress = currentIdx
	}

	ignore0 := false
	for pieceIdx := 0; pieceIdx < 8; pieceIdx++ {
		if ignore0 && address[pieceIdx] == 0 {
			continue
		} else if ignore0 {
			ignore0 = false
		}
		if compress == pieceIdx {
			separator := ":"
			if pieceIdx == 0 {
				separator = "::"
			}
			output += separator
			ignore0 = true
			continue
		}
		output += strconv.FormatUint(uint64(address[pieceIdx]), 16)
		// 32512
		if pieceIdx != 7 {
			output += ":"
		}
	}
	return output
}

type IPv4Addr uint32

func (address IPv4Addr) String() string {
	return strconv.Itoa(int(address>>24)) + "." +
		strconv.Itoa(int((address>>16)&0xFF)) + "." +
		strconv.Itoa(int((address>>8)&0xFF)) + "." +
		strconv.Itoa(int(address&0xFF))
}

var idnaProfile = idna.New(
	idna.MapForLookup(),
	idna.BidiRule(),
	idna.VerifyDNSLength(false),
	idna.StrictDomainName(true),
	idna.ValidateLabels(true),
	idna.CheckHyphens(false),
	idna.CheckJoiners(true),
	idna.Transitional(false),
)

// ToASCII converts a string to ASCII using IDNA
// https://url.spec.whatwg.org/#concept-domain-to-ascii
func (p *parser) ToASCII(src string, beStrict bool) (string, error) {
	if src == "" {
		return "", nil
	}

	// If encoding is set, convert to Unicode
	if p.opts.encodingOverride != nil {
		if u, err := p.stringToUnicode(src); err == nil {
			src = u
		}
	}

	// Convert to punycode
	a, err := idnaProfile.ToASCII(src)
	if err != nil {
		if !beStrict {
			if containsOnlyASCIIOrMiscAndNoPunycode(src) {
				return a, nil
			}
		}

		if !p.opts.laxHostParsing {
			return a, err
		}
	}
	if a == "" {
		return "", fmt.Errorf("idna toAscii returned empty string")
	}
	return a, nil
}

// containsOnlyASCIIOrMiscAndNoPunycode returns true if the string contains only ASCII characters or characters from Section 4.1.1 in UTS #46
// and does not contain any labels starting with acePrefix (xn--)
func containsOnlyASCIIOrMiscAndNoPunycode(s string) bool {
	s = strings.ToLower(s)
	p := 0
	for _, r := range s {
		if r >= utf8.RuneSelf && r != '\u2260' && r != '\u226e' && r != '\u226f' {
			return false
		}
		switch {
		case r == '.':
			p = 0
		case p == 0 && r == 'x':
			p = 1
		case p == 1 && r == 'n':
			p = 2
		case p == 2 && r == '-':
			p = 3
		case p == 3 && r == '-':
			return false
		default:
			p = -1
		}
	}
	return true
}

func (p *parser) stringToUnicode(src string) (string, error) {
	var bb []byte
	for _, r := range src {
		if b, ok := p.opts.encodingOverride.EncodeRune(r); ok && b > 31 {
			bb = append(bb, b)
		} else {
			return "", fmt.Errorf("could not conver %v to Unicode using %v", src, p.opts.encodingOverride.String())
		}
	}
	return string(bb), nil
}

func percentEncodeString(s string, tr *PercentEncodeSet) string {
	sb := strings.Builder{}
	for _, b := range []byte(s) {
		sb.WriteString(percentEncodeByte(b, tr))
	}
	return sb.String()
}

func percentEncodeByte(b byte, tr *PercentEncodeSet) string {
	if tr != nil && !tr.ByteShouldBeEncoded(b) {
		return string(b)
	}

	percentEncoded := make([]byte, 3)
	percentEncoded[0] = '%'
	percentEncoded[1] = "0123456789ABCDEF"[b>>4]
	percentEncoded[2] = "0123456789ABCDEF"[b&15]
	return string(percentEncoded)
}
