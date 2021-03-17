package caddy

import (
	"fmt"
	"strings"
)

const tabSpaces = 2

// ParsedString is the representation of the parser
type ParsedString struct {
	openedDelimiters  []string
	waitingDelimiters []string
	str               string
}

// NewParser will create a new parser to parse Caddyfile configuration
func NewParser() *ParsedString {
	return &ParsedString{
		openedDelimiters:  []string{},
		waitingDelimiters: []string{},
		str:               "",
	}
}

func getDelimiters() map[string]string {
	return map[string]string{
		"[": "]",
		"{": "}",
	}
}

func getCurrentNature(s string) (isDelimiter bool, isOpening bool) {
	for k, v := range getDelimiters() {
		if s == k {
			return true, true
		}
		if s == v {
			return true, false
		}
	}
	return false, false
}

func pop(delimiter []string) []string {
	return append(delimiter[:len(delimiter)-1])
}

func (p *ParsedString) updateDelimiter(s string, isOpening bool) {
	if isOpening {
		if len(p.openedDelimiters) > 0 && p.openedDelimiters[len(p.openedDelimiters) - 1] == "[" {
			p.appendContent("", "")
		}
		p.openedDelimiters = append(p.openedDelimiters, s)
		p.waitingDelimiters = append(p.waitingDelimiters, getDelimiters()[s])
	} else if s == p.waitingDelimiters[len(p.waitingDelimiters) - 1] {
		p.openedDelimiters = pop(p.openedDelimiters)
		p.waitingDelimiters = pop(p.waitingDelimiters)
	}
}

func (p *ParsedString) appendContent(s1 string, s2 string) {
	lineValue := ""
	odLen := len(p.openedDelimiters)
	if odLen > 0 && p.openedDelimiters[odLen - 1] == "[" {
		lineValue += "- "
	}
	lineValue += s1

	isDelimiter, isOpening := getCurrentNature(s2)

	if isDelimiter || s1 != s2 || odLen < 1 || (odLen > 0 && p.openedDelimiters[odLen - 1] != "[") {
		lineValue += ":"
	}
	if s1 != s2 && !isDelimiter {
		lineValue += fmt.Sprintf(" %s", s2)
	}
	p.str = fmt.Sprintf("%s%s%s\n", p.str, strings.Repeat(" ", tabSpaces* len(p.openedDelimiters)), lineValue)
	if isDelimiter {
		p.updateDelimiter(s2, isOpening)
	}
}

// WriteLine will write the yaml formatted line
func (p *ParsedString) WriteLine(s1 string, s2 string) {
	isDelimiter, isOpening := getCurrentNature(s1)

	if isDelimiter {
		p.updateDelimiter(s1, isOpening)
	} else {
		p.appendContent(s1, s2)
	}
}
