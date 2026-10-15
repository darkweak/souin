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

package canonicalizer

import (
	"strings"

	"github.com/nlnwa/whatwg-url/errors"
	"github.com/nlnwa/whatwg-url/url"
)

func New(opts ...url.ParserOption) url.Parser {
	p := &profile{
		Parser:    url.NewParser(opts...),
		sortQuery: NoSort,
	}
	for _, opt := range opts {
		if o, ok := opt.(canonParserOption); ok {
			o.applyProfile(p)
		}
	}
	return p
}

type profile struct {
	url.Parser
	removeUserInfo          bool
	removePort              bool
	removeFragment          bool
	sortQuery               querySort
	repeatedPercentDecoding bool
	defaultScheme           string
}

func (p *profile) Parse(rawUrl string) (*url.Url, error) {
	u, err := p.Parser.Parse(rawUrl)
	if err != nil {
		if errors.Type(err) == errors.MissingSchemeNonRelativeURL && p.defaultScheme != "" {
			rawUrl = p.defaultScheme + "://" + rawUrl
			u, err = p.Parser.Parse(rawUrl)
		}
		if err != nil {
			return nil, err
		}
	}

	return p.Canonicalize(u)
}

func (p *profile) ParseRef(rawUrl, ref string) (*url.Url, error) {
	b, err := p.Parser.Parse(rawUrl)
	if err != nil {
		if errors.Type(err) == errors.MissingSchemeNonRelativeURL && p.defaultScheme != "" {
			rawUrl = p.defaultScheme + "://" + rawUrl
			b, err = p.Parser.Parse(rawUrl)
		}
		if err != nil {
			return nil, err
		}
	}

	u, err := b.Parse(ref)
	if err != nil {
		return nil, err
	}

	return p.Canonicalize(u)
}

func (p *profile) Canonicalize(u *url.Url) (*url.Url, error) {
	if p.repeatedPercentDecoding {
		if u.Hostname() != "" {
			u.SetHostname(decodeEncode(u.Hostname(), url.HostPercentEncodeSet))
		}
		if u.Pathname() != "" {
			u.SetPathname(decodeEncode(u.Pathname(), LaxPathPercentEncodeSet))
		}
		if u.Search() != "" {
			u.SearchParams().Iterate(func(pair *url.NameValuePair) {
				pair.Name = decodeEncode(pair.Name, RepeatedQueryPercentDecodeSet)
				pair.Value = decodeEncode(pair.Value, RepeatedQueryPercentDecodeSet)
			})
		}
		if u.Hash() != "" {
			u.SetHash(decodeEncode(strings.TrimPrefix(u.Hash(), "#"), url.HostPercentEncodeSet))
		}
	}

	if p.removePort {
		u.SetPort("")
	}
	if p.removeUserInfo {
		u.SetUsername("")
		u.SetPassword("")
	}
	if p.removeFragment {
		u.SetHash("")
	}

	switch p.sortQuery {
	case SortKeys:
		u.SearchParams().Sort()
	case SortParameter:
		u.SearchParams().SortAbsolute()
	}

	return u, nil
}

func decodeEncode(s string, tr *url.PercentEncodeSet) string {
	r := percentEncode(repeatedDecode(s), tr)
	return r
}

// repeatedDecode repeatedly percent-unescape a string until it has no more percent-escapes
func repeatedDecode(s string) string {
	var r string
	for {
		r = decodePercentEncoded(s)
		if s == r {
			break
		}
		s = r
	}
	return s
}

func percentEncode(s string, tr *url.PercentEncodeSet) string {
	sb := strings.Builder{}
	for _, b := range []byte(s) {
		sb.WriteString(percentEncodeByte(b, tr.Set('%')))
	}
	return sb.String()
}

func percentEncodeByte(b byte, tr *url.PercentEncodeSet) string {
	if tr != nil && !tr.ByteShouldBeEncoded(b) {
		return string(b)
	}

	percentEncoded := make([]byte, 3)
	percentEncoded[0] = '%'
	percentEncoded[1] = "0123456789ABCDEF"[b>>4]
	percentEncoded[2] = "0123456789ABCDEF"[b&15]
	return string(percentEncoded)
}

func decodePercentEncoded(s string) string {
	sb := strings.Builder{}
	bytes := []byte(s)
	for i := 0; i < len(bytes); i++ {
		if bytes[i] != '%' {
			sb.WriteByte(bytes[i])
		} else if len(bytes) < (i+3) ||
			(!url.ASCIIHexDigit.Test(uint(bytes[i+1])) || !url.ASCIIHexDigit.Test(uint(bytes[i+2]))) {
			sb.WriteByte(bytes[i])
		} else {
			b := unhex(bytes[i+1])<<4 | unhex(bytes[i+2])
			sb.WriteByte(b)
			i += 2
		}
	}
	return sb.String()
}

func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}
