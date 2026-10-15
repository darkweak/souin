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
	"strings"
	"unicode/utf8"
)

type inputString struct {
	s       string
	runes   []rune
	pointer int
	eof     bool
	length  int
}

func newInputString(s string) *inputString {
	i := &inputString{runes: []rune(s), pointer: -1}
	i.s = s
	i.length = len(i.runes)
	return i
}

func (i *inputString) nextCodePoint() rune {
	i.pointer++
	if i.pointer >= i.length {
		i.eof = true
		return utf8.RuneError
	}
	r := i.runes[i.pointer]

	return r
}

func (i *inputString) currentIsInvalid() bool {
	return i.runes[i.pointer] == utf8.RuneError
}

func (i *inputString) getCurrentAsByte() byte {
	if i.pointer >= i.length {
		i.eof = true
		return 0
	}
	var pos int
	for j := 0; j < i.pointer; j++ {
		pos += utf8.RuneLen(i.runes[j])
	}
	return i.s[pos]
}

func (i *inputString) rewindLast() {
	i.eof = false
	i.pointer--
}

func (i *inputString) reset() {
	i.pointer = -1
	i.eof = false
}

func (i *inputString) rewind(length int) {
	i.pointer -= length
	i.eof = false
}

func (i *inputString) remainingFromPointer() string {
	if i.eof {
		return ""
	}
	return string(i.runes[i.pointer:])
}

func (i *inputString) remainingStartsWith(s string) bool {
	if i.eof {
		return false
	}

	return strings.HasPrefix(string(i.runes[i.pointer+1:]), s)
}

// remainingIsInvalidPercentEncoded returns true if the first three characters in the rune array are not '%' followed by two hex digits.
// If true, the second return value is the invalid percent encoded string.
func (i *inputString) remainingIsInvalidPercentEncoded() (bool, string) {
	return remainingIsInvalidPercentEncoded(i.runes[i.pointer:])
}

// remainingIsInvalidPercentEncoded returns true if the first three characters in the rune array are not '%' followed by two hex digits.
// If true, the second return value is the invalid percent encoded string.
func remainingIsInvalidPercentEncoded(runes []rune) (bool, string) {
	if len(runes) >= 1 && runes[0] == '%' &&
		(len(runes) < 3 || (!ASCIIHexDigit.Test(uint(runes[1])) || !ASCIIHexDigit.Test(uint(runes[2])))) {
		l := 3
		if len(runes) < 3 {
			l = len(runes)
		}
		return true, string(runes[0:l])
	}
	return false, ""
}

func (i *inputString) String() string {
	return string(i.runes)
}
