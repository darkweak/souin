package esi

import (
	"net/http"
)

func findTagName(b []byte) Tag {
	name := tagname.FindSubmatch(b)
	if name == nil {
		return nil
	}

	switch string(name[1]) {
	case comment:
		return &commentTag{
			baseTag: newBaseTag(),
		}
	case choose:
		return &chooseTag{
			baseTag: newBaseTag(),
		}
	case escape:
		return &escapeTag{
			baseTag: newBaseTag(),
		}
	case include:
		return &includeTag{
			baseTag: newBaseTag(),
		}
	case remove:
		return &removeTag{
			baseTag: newBaseTag(),
		}
	case try:
	case vars:
		return &varsTag{
			baseTag: newBaseTag(),
		}
	default:
		return nil
	}

	return nil
}

func HasOpenedTags(b []byte) bool {
	return esi.FindIndex(b) != nil || escapeRg.FindIndex(b) != nil
}

func CanProcess(b []byte) bool {
	if tag := findTagName(b); tag != nil {
		return tag.HasClose(b)
	}

	return false
}

func ReadToTag(next []byte, pointer int) (startTagPosition, esiPointer int, t Tag) {
	tagIdx := esi.FindIndex(next)
	var isEscapeTag bool

	if escIdx := escapeRg.FindIndex(next); escIdx != nil && (tagIdx == nil || escIdx[0] < tagIdx[0]) {
		tagIdx = escIdx
		tagIdx[1] = escIdx[0]
		isEscapeTag = true
	}

	if tagIdx == nil {
		return len(next), 0, nil
	}

	esiPointer = tagIdx[1]
	startTagPosition = tagIdx[0]
	t = findTagName(next[esiPointer:])

	if isEscapeTag {
		esiPointer += 7
	}

	return
}

func Parse(b []byte, req *http.Request) []byte {
	pointer := 0

	for pointer < len(b) {
		var escapeTag bool

		next := b[pointer:]
		tagIdx := esi.FindIndex(next)

		if escIdx := escapeRg.FindIndex(next); escIdx != nil && (tagIdx == nil || escIdx[0] < tagIdx[0]) {
			tagIdx = escIdx
			tagIdx[1] = escIdx[0]
			escapeTag = true
		}

		if tagIdx == nil {
			break
		}

		esiPointer := tagIdx[1]
		t := findTagName(next[esiPointer:])

		if escapeTag {
			esiPointer += 7
		}

		res, p := t.Process(next[esiPointer:], req)
		esiPointer += p

		b = append(b[:pointer], append(next[:tagIdx[0]], append(res, next[esiPointer:]...)...)...)
		pointer += len(res) + tagIdx[0]
	}

	return b
}
