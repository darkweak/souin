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

import "golang.org/x/text/encoding/charmap"

var defaultSpecialSchemes = map[string]string{
	"ftp":   "21",
	"file":  "",
	"http":  "80",
	"https": "443",
	"ws":    "80",
	"wss":   "443",
}

// parserOptions configure a url parser. parserOptions are set by the ParserOption
// values passed to NewParser.
type parserOptions struct {
	reportValidationErrors              bool
	failOnValidationError               bool
	laxHostParsing                      bool
	collapseConsecutiveSlashes          bool
	acceptInvalidCodepoints             bool
	preParseHostFunc                    func(url *Url, host string) string
	postParseHostFunc                   func(url *Url, host string) string
	percentEncodeSinglePercentSign      bool
	allowSettingPathForNonBaseUrl       bool
	skipWindowsDriveLetterNormalization bool
	specialSchemes                      map[string]string
	skipTrailingSlashNormalization      bool
	encodingOverride                    *charmap.Charmap
	pathPercentEncodeSet                *PercentEncodeSet
	specialQueryPercentEncodeSet        *PercentEncodeSet
	queryPercentEncodeSet               *PercentEncodeSet
	specialFragmentPercentEncodeSet     *PercentEncodeSet
	fragmentPercentEncodeSet            *PercentEncodeSet
	skipEqualsForEmptySearchParamsValue bool
}

// ParserOption configures how we parse a URL.
type ParserOption interface {
	apply(*parserOptions)
}

// EmptyParserOption does not alter the parser configuration. It can be embedded in
// another structure to build custom parser options.
type EmptyParserOption struct{}

func (EmptyParserOption) apply(*parserOptions) {}

// funcParserOption wraps a function that modifies parserOptions into an
// implementation of the ParserOption interface.
type funcParserOption struct {
	f func(*parserOptions)
}

func (fpo *funcParserOption) apply(po *parserOptions) {
	fpo.f(po)
}

func newFuncParserOption(f func(*parserOptions)) *funcParserOption {
	return &funcParserOption{
		f: f,
	}
}

func defaultParserOptions() parserOptions {
	return parserOptions{
		pathPercentEncodeSet:            PathPercentEncodeSet,
		specialQueryPercentEncodeSet:    SpecialQueryPercentEncodeSet,
		queryPercentEncodeSet:           QueryPercentEncodeSet,
		specialFragmentPercentEncodeSet: FragmentPercentEncodeSet,
		fragmentPercentEncodeSet:        FragmentPercentEncodeSet,
		specialSchemes:                  defaultSpecialSchemes,
	}
}

// WithReportValidationErrors records all non fatal validation errors so that they can be fetchd by a call to....
func WithReportValidationErrors() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.reportValidationErrors = true
	})
}

// WithFailOnValidationError makes the parser throw an error on non fatal validation errors.
func WithFailOnValidationError() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.failOnValidationError = true
	})
}

// WithLaxHostParsing ignores some decoding errors and returns the host as is.
//
// This API is EXPERIMENTAL.
func WithLaxHostParsing() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.laxHostParsing = true
	})
}

// WithCollapseConsecutiveSlashes collapses consecutive slashes in path into one
// (e.g. http://example.com//foo///bar => http://example.com/foo/bar).
func WithCollapseConsecutiveSlashes() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.collapseConsecutiveSlashes = true
	})
}

// WithAcceptInvalidCodepoints percent encodes values which are not valid UTF-8.
//
// This API is EXPERIMENTAL.
func WithAcceptInvalidCodepoints() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.acceptInvalidCodepoints = true
	})
}

// WithPreParseHostFunc is a function which allows manipulation of host string before it is parsed.
//
// This API is EXPERIMENTAL.
func WithPreParseHostFunc(f func(url *Url, host string) string) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.preParseHostFunc = f
	})
}

// WithPostParseHostFunc is a function which allows manipulation of host string after it is parsed.
// It is called only if the host isn't an IP address.
//
// This API is EXPERIMENTAL.
func WithPostParseHostFunc(f func(url *Url, host string) string) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.postParseHostFunc = f
	})
}

// WithPercentEncodeSinglePercentSign percent encodes a '%' which is not followed by two hexadecimal digits
// instead of complaining about invalid percent encoding.
//
// This API is EXPERIMENTAL.
func WithPercentEncodeSinglePercentSign() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.percentEncodeSinglePercentSign = true
	})
}

// WithAllowSettingPathForNonBaseUrl allows to set path for a url which cannot be a base url.
// WhathWg standard says this should be illegal
//
// This API is EXPERIMENTAL.
func WithAllowSettingPathForNonBaseUrl() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.allowSettingPathForNonBaseUrl = true
	})
}

// WithSkipWindowsDriveLetterNormalization skips conversion of 'C|' to 'C:'.
// WhathWg standard says only a normalized Windows drive letter is conforming.
//
// This API is EXPERIMENTAL.
func WithSkipWindowsDriveLetterNormalization() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.skipWindowsDriveLetterNormalization = true
	})
}

// WithSpecialSchemes allows overriding the notion of special schemes.
// special is a map of 'scheme' => 'default port'
//
// WhatWg standard removed gopher from special schemes. This is how you add it back:
//
//	special := map[string]string{
//	                       "ftp":    "21",
//	                       "file":   "",
//	                       "http":   "80",
//	                       "https":  "443",
//	                       "ws":     "80",
//	                       "wss":    "443",
//	                       "gopher": "70",
//	               }
//
// This API is EXPERIMENTAL.
func WithSpecialSchemes(special map[string]string) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.specialSchemes = special
	})
}

// WithEncodingOverride allows to set an encoding other than UTF-8 when parsing.
//
// This API is EXPERIMENTAL.
func WithEncodingOverride(cm *charmap.Charmap) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.encodingOverride = cm
	})
}

// WithPathPercentEncodeSet allows to set an alternative set of characters to percent encode in path component.
//
// This API is EXPERIMENTAL.
func WithPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.pathPercentEncodeSet = encodeSet
	})
}

// WithQueryPercentEncodeSet allows to set an alternative set of characters to percent encode in query component
// when scheme is not special.
//
// This API is EXPERIMENTAL.
func WithQueryPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.queryPercentEncodeSet = encodeSet
	})
}

// WithSpecialQueryPercentEncodeSet allows to set an alternative set of characters to percent encode in query component
// when scheme is special.
//
// This API is EXPERIMENTAL.
func WithSpecialQueryPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.specialQueryPercentEncodeSet = encodeSet
	})
}

// WithFragmentPathPercentEncodeSet allows to set an alternative set of characters to percent encode in fragment
// component when scheme is not special.
//
// This API is EXPERIMENTAL.
func WithFragmentPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.fragmentPercentEncodeSet = encodeSet
	})
}

// WithSpecialFragmentPathPercentEncodeSet allows to set an alternative set of characters to percent encode in fragment
// component when scheme is special.
//
// This API is EXPERIMENTAL.
func WithSpecialFragmentPathPercentEncodeSet(encodeSet *PercentEncodeSet) ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.fragmentPercentEncodeSet = encodeSet
	})
}

// WithSkipTrailingSlashNormalization skips normalizing of empty paths.
//
// This API is EXPERIMENTAL.
func WithSkipTrailingSlashNormalization() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.skipTrailingSlashNormalization = true
	})
}

// WithSkipEqualsForEmptySearchParamsValue skips writing '=' when setting an empty value for a search parameter.
//
// e.g. url.SearchParams().Set("name", "") gives 'http://...?name' instead of 'http://...?name='
//
// This API is EXPERIMENTAL.
func WithSkipEqualsForEmptySearchParamsValue() ParserOption {
	return newFuncParserOption(func(o *parserOptions) {
		o.skipEqualsForEmptySearchParamsValue = true
	})
}
