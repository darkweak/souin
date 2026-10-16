package canonicalizer

import "github.com/nlnwa/whatwg-url/url"

// canonParserOption configures how we canonicalize a URL.
type canonParserOption interface {
	applyProfile(*profile)
}

// funcCanonParserOption wraps a function that modifies profile into an
// implementation of both the url.ParserOption and canonParserOption interface.
type funcCanonParserOption struct {
	url.EmptyParserOption
	f func(*profile)
}

func (cpo *funcCanonParserOption) applyProfile(p *profile) {
	cpo.f(p)
}

// WithRemoveUserInfo removes username and password from url.
//
// This API is EXPERIMENTAL.
func WithRemoveUserInfo() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.removeUserInfo = true
		},
	}
}

// WithRemovePort always removes port from url. Default is to remove port if port is default port for scheme.
//
// This API is EXPERIMENTAL.
func WithRemovePort() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.removePort = true
		},
	}
}

// WithRemoveFragment removes the fragment part of the url.
//
// This API is EXPERIMENTAL.
func WithRemoveFragment() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.removeFragment = true
		},
	}
}

// WithRepeatedPercentDecoding.
//
// This API is EXPERIMENTAL.
func WithRepeatedPercentDecoding() url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.repeatedPercentDecoding = true
		},
	}
}

// WithDefaultScheme sets a scheme to add if url is missing scheme.
//
// This API is EXPERIMENTAL.
func WithDefaultScheme(scheme string) url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.defaultScheme = scheme
		},
	}
}

// WithSortQuery sets sort type for query parameters.
// if query should be sorted: 0 = no sort, 1 = sort keys, but leave repeated keys in same order, 2 = sort key,value
//
// This API is EXPERIMENTAL.
func WithSortQuery(sortType querySort) url.ParserOption {
	return &funcCanonParserOption{
		f: func(p *profile) {
			p.sortQuery = sortType
		},
	}
}

type querySort int

const (
	// Do not sort query.
	NoSort querySort = iota
	// Stable sort on query parameter keys.
	SortKeys
	// Sort on entire query parameter.
	SortParameter
)
