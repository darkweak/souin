package url

import (
	"strings"
)

type path struct {
	p      []string
	opaque bool
}

func (p *path) isOpaque() bool {
	return p.opaque
}

func (p *path) isEmpty() bool {
	return len(p.p) == 0
}

func (p *path) setOpaque(opaquePath string) {
	p.p = []string{opaquePath}
	p.opaque = true
}

func (p *path) addSegment(segment string) {
	p.p = append(p.p, segment)
	p.opaque = false
}

func (p *path) init() {
	p.p = []string{}
	p.opaque = false
}

func (p *path) shortenPath(scheme string) {
	if scheme == "file" && len(p.p) == 1 && isNormalizedWindowsDriveLetter(p.p[0]) {
		return
	}
	if len(p.p) == 0 {
		return
	}
	p.p = p.p[0 : len(p.p)-1]
}

func (p *path) stripTrailingSpacesIfOpaque() {
	if p.opaque {
		p.p[0] = strings.TrimRight(p.p[0], "\u0020")
	}
}

func (p *path) clone() *path {
	if p == nil {
		return nil
	}
	newPath := &path{
		opaque: p.opaque,
	}
	if p.p != nil {
		newPath.p = make([]string, len(p.p))
		copy(newPath.p, p.p)
	}
	return newPath
}

func (p *path) String() string {
	if p.opaque {
		return p.p[0]
	} else {
		output := ""
		for _, pp := range p.p {
			output += "/" + pp
		}
		return output
	}
}
