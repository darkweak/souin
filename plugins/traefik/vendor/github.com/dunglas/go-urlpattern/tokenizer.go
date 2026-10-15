package urlpattern

import (
	"errors"
	"fmt"
	"unicode"

	"golang.org/x/exp/utf8string"
)

var ErrType = errors.New("type error")

// https://wicg.github.io/urlpattern/#tokenizing
type tokenizePolicy bool

const (
	tokenizePolicyLenient tokenizePolicy = false
	tokenizePolicyStrict  tokenizePolicy = true
)

type tokenizer struct {
	input     *utf8string.String
	policy    tokenizePolicy
	tokenList []token
	index     int
	nextIndex int
	codePoint rune
}

func tokenize(input string, policy tokenizePolicy) ([]token, error) {
	t := tokenizer{
		input:     utf8string.NewString(input),
		policy:    policy,
		tokenList: make([]token, 0, len(input)),
	}

	len := t.input.RuneCount()

	for t.index < len {
		t.seekAndGetNextCodePoint(t.index)

		switch t.codePoint {
		case '*':
			t.addTokenWithDefaultPositionAndLength(tokenAsterisk)

		case '+':
			t.addTokenWithDefaultPositionAndLength(tokenOtherModifier)

		case '?':
			t.addTokenWithDefaultPositionAndLength(tokenOtherModifier)

		case '\\':
			if t.index == len-1 {
				if err := t.processTokenizingError(t.nextIndex, t.index); err != nil {
					return nil, err
				}

				continue
			}

			escapedIndex := t.nextIndex
			t.getNextCodePoint()
			t.addTokenWithDefaultLength(tokenEscapedChar, t.nextIndex, escapedIndex)

		case '{':
			t.addTokenWithDefaultPositionAndLength(tokenOpen)

		case '}':
			t.addTokenWithDefaultPositionAndLength(tokenClose)

		case ':':
			namePosition := t.nextIndex
			nameStart := namePosition

			for namePosition < len {
				t.seekAndGetNextCodePoint(namePosition)

				var firstCodePoint bool
				if namePosition == nameStart {
					firstCodePoint = true
				}

				if !isValidNameCodePoint(t.codePoint, firstCodePoint) {
					break
				}

				namePosition = t.nextIndex
			}

			if namePosition <= nameStart {
				if err := t.processTokenizingError(nameStart, t.index); err != nil {
					return nil, err
				}

				continue
			}

			t.addTokenWithDefaultLength(tokenName, namePosition, nameStart)

		case '(':
			depth := 1
			regexpPosition := t.nextIndex
			regexpStart := regexpPosition
			err := false

		Loop:
			for regexpPosition < len {
				t.seekAndGetNextCodePoint(regexpPosition)
				if !isASCII(t.codePoint) ||
					(regexpPosition == regexpStart && t.codePoint == '?') {
					if e := t.processTokenizingError(regexpStart, t.index); e != nil {
						return nil, e
					}

					err = true
					break
				}

				switch t.codePoint {
				case '\\':
					if regexpPosition == len-1 {
						if e := t.processTokenizingError(regexpStart, t.index); e != nil {
							return nil, e
						}

						err = true
						break Loop
					}

					t.getNextCodePoint()

					if !isASCII(t.codePoint) {
						if e := t.processTokenizingError(regexpStart, t.index); e != nil {
							return nil, e
						}

						err = true
						break Loop
					}

					regexpPosition = t.nextIndex

					continue

				case ')':
					depth--
					if depth == 0 {
						regexpPosition = t.nextIndex
						break Loop
					}

				case '(':
					depth++

					if regexpPosition == len-1 {
						if e := t.processTokenizingError(regexpStart, t.index); e != nil {
							return nil, e
						}

						err = true
						break Loop
					}

					temporaryPosition := t.nextIndex
					t.getNextCodePoint()

					if t.codePoint != '?' {
						if e := t.processTokenizingError(regexpStart, t.index); e != nil {
							return nil, e
						}

						err = true
						break Loop
					}

					t.nextIndex = temporaryPosition
				}

				regexpPosition = t.nextIndex
			}

			if err {
				continue
			}

			if depth != 0 {
				if e := t.processTokenizingError(regexpStart, t.index); e != nil {
					return nil, e
				}

				continue
			}

			regexpLength := regexpPosition - regexpStart - 1
			if regexpLength == 0 {
				if e := t.processTokenizingError(regexpStart, t.index); e != nil {
					return nil, e
				}

				continue
			}

			t.addToken(tokenRegexp, regexpPosition, regexpStart, regexpLength)

		default:
			t.addTokenWithDefaultPositionAndLength(tokenChar)
		}
	}

	t.addTokenWithDefaultLength(tokenEnd, t.index, t.index)

	return t.tokenList, nil
}

func (t *tokenizer) getNextCodePoint() {
	t.codePoint = t.input.At(t.nextIndex)
	t.nextIndex++
}

func (t *tokenizer) seekAndGetNextCodePoint(index int) {
	t.nextIndex = index
	t.getNextCodePoint()
}

func (t *tokenizer) addToken(tType tokenType, nextPosition, valuePosition, valueLength int) {
	t.tokenList = append(t.tokenList, token{
		tType: tType,
		index: t.index,
		value: t.input.Slice(valuePosition, valuePosition+valueLength),
	})
	t.index = nextPosition
}

func (t *tokenizer) addTokenWithDefaultLength(tType tokenType, nextPosition, valuePosition int) {
	t.addToken(tType, nextPosition, valuePosition, nextPosition-valuePosition)
}

func (t *tokenizer) addTokenWithDefaultPositionAndLength(tType tokenType) {
	t.addTokenWithDefaultLength(tType, t.nextIndex, t.index)
}

func (t *tokenizer) processTokenizingError(nextPosition, valuePosition int) error {
	if t.policy == tokenizePolicyStrict {
		return fmt.Errorf("%w: %#v", ErrType, t)
	}

	t.addTokenWithDefaultLength(tokenInvalidChar, nextPosition, valuePosition)

	return nil
}

func isValidNameCodePoint(codePoint rune, first bool) bool {
	if first {
		return isIdentifierStart(codePoint)
	}

	return isIdentifierPart(codePoint)
}

func isIdentifierStart(codePoint rune) bool {
	return unicode.In(
		codePoint,
		unicode.L,
		unicode.Nl,
		unicode.Other_ID_Start,
	) && !unicode.In(
		codePoint,
		unicode.Pattern_Syntax,
		unicode.Pattern_White_Space,
	)
}

func isIdentifierPart(codePoint rune) bool {
	return unicode.In(
		codePoint,
		unicode.L,
		unicode.Nl,
		unicode.Other_ID_Start,
		unicode.Mn,
		unicode.Mc,
		unicode.Nd,
		unicode.Pc,
		unicode.Other_ID_Continue,
	) && !unicode.In(
		codePoint,
		unicode.Pattern_Syntax,
		unicode.Pattern_White_Space,
	)
}

func isASCII(codePoint rune) bool {
	return codePoint >= 0 && codePoint <= unicode.MaxASCII
}
