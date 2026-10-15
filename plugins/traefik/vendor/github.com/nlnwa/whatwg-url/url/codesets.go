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
	"unicode"

	"github.com/bits-and-blooms/bitset"
)

type PercentEncodeSet struct {
	bs       *bitset.BitSet
	allBelow int32
}

func NewPercentEncodeSet(allBelow int32, bytes ...uint) *PercentEncodeSet {
	p := &PercentEncodeSet{allBelow: allBelow, bs: bitset.New(0x7f)}
	for _, b := range bytes {
		p.bs.Set(b)
	}
	return p
}

func (p *PercentEncodeSet) Set(bytes ...uint) *PercentEncodeSet {
	r := &PercentEncodeSet{
		allBelow: p.allBelow,
		bs:       p.bs.Clone(),
	}
	for _, b := range bytes {
		r.bs.Set(b)
	}
	return r
}

func (p *PercentEncodeSet) Clear(bytes ...uint) *PercentEncodeSet {
	r := &PercentEncodeSet{
		allBelow: p.allBelow,
		bs:       p.bs.Clone(),
	}
	for _, b := range bytes {
		r.bs.Clear(b)
	}
	return r
}

func (p *PercentEncodeSet) RuneShouldBeEncoded(r rune) bool {
	if r < p.allBelow || r > 0x007E || p.bs.Test(uint(r)) {
		return true
	}
	return false
}

func (p *PercentEncodeSet) ByteShouldBeEncoded(b byte) bool {
	if int32(b) < p.allBelow || b > 0x007E || p.bs.Test(uint(b)) {
		return true
	}
	return false
}

func (p *PercentEncodeSet) RuneNotInSet(r rune) bool {
	if r < p.allBelow || p.bs.Test(uint(r)) {
		return false
	}
	return true
}

func isURLCodePoint(r rune) bool {
	if ASCIIAlphanumeric.Test(uint(r)) {
		return true
	}
	if someURLCodePoints.Test(uint(r)) {
		return true
	}
	if r >= 0xa0 && r <= 0x10fffd {
		if unicode.Is(unicode.Noncharacter_Code_Point, r) {
			return false
		}
		if unicode.Is(unicode.Cs, r) {
			return false
		}
		return true
	}
	return false
}

var ASCIITabOrNewline = bitset.New(0x0d).Set(0x09).Set(0x0a).Set(0x0d)
var ASCIIAlpha = bitset.New(0x7a)
var ASCIIDigit = bitset.New(0x39)
var ASCIIHexDigit = bitset.New(0x66)
var ASCIIAlphanumeric = bitset.New(0x7a)
var C0control = bitset.New(0x1f)
var C0controlOrSpace = bitset.New(0x20).Set(0x20)
var ForbiddenHostCodePoint = bitset.New(0x7c).Set(0x00).Set(0x09).Set(0x0a).Set(0x0d).Set(0x20).
	Set(0x23).Set(0x2f).Set(0x3a).Set(0x3c).Set(0x3e).Set(0x3f).Set(0x40).Set(0x5b).
	Set(0x5c).Set(0x5d).Set(0x5e).Set(0x7c)
var ForbiddenDomainCodePoint = ForbiddenHostCodePoint.Clone().Set(0x25).Set(0x7f)
var someURLCodePoints = bitset.New(0x7e).Set(0x24).Set(0x26).Set(0x27).Set(0x28).Set(0x29).
	Set(0x2a).Set(0x2b).Set(0x2c).Set(0x2d).Set(0x2e).Set(0x2f).Set(0x3a).Set(0x3b).Set(0x3d).
	Set(0x3f).Set(0x40).Set(0x5f).Set(0x7e)

var C0PercentEncodeSet = NewPercentEncodeSet(0x20)
var C0OrSpacePercentEncodeSet = NewPercentEncodeSet(0x21)
var FragmentPercentEncodeSet = C0OrSpacePercentEncodeSet.Set(0x22, 0x3c, 0x3e, 0x60)
var QueryPercentEncodeSet = C0OrSpacePercentEncodeSet.Set(0x22, 0x23, 0x3C, 0x3E)
var SpecialQueryPercentEncodeSet = QueryPercentEncodeSet.Set(0x27)
var PathPercentEncodeSet = QueryPercentEncodeSet.Set(0x3f, 0x60, 0x7b, 0x7d)
var UserInfoPercentEncodeSet = PathPercentEncodeSet.Set(0x2f, 0x3a, 0x3b, 0x3d, 0x40, 0x5b, 0x5c, 0x5d, 0x5e, 0x7c)
var HostPercentEncodeSet = C0OrSpacePercentEncodeSet.Set(0x23)

func init() {
	for i := 'a'; i <= 'z'; i++ {
		ASCIIAlpha.Set(uint(i))
	}
	for i := 'A'; i <= 'Z'; i++ {
		ASCIIAlpha.Set(uint(i))
	}

	for i := '0'; i <= '9'; i++ {
		ASCIIDigit.Set(uint(i))
	}

	ASCIIAlphanumeric.InPlaceUnion(ASCIIAlpha)
	ASCIIAlphanumeric.InPlaceUnion(ASCIIDigit)

	ASCIIHexDigit.InPlaceUnion(ASCIIDigit)
	for i := 'A'; i <= 'F'; i++ {
		ASCIIHexDigit.Set(uint(i))
	}
	for i := 'a'; i <= 'f'; i++ {
		ASCIIHexDigit.Set(uint(i))
	}

	// Add C0 control characters
	for i := 0x00; i <= 0x1f; i++ {
		C0control.Set(uint(i))
		C0controlOrSpace.Set(uint(i))
		ForbiddenDomainCodePoint.Set(uint(i))
	}
}
