package rfc

import (
	"net/http"
	"strings"
	"time"

	"github.com/darkweak/storages/core"
)

type Revalidator struct {
	Matched                     bool
	IfNoneMatchPresent          bool
	IfMatchPresent              bool
	IfModifiedSincePresent      bool
	IfUnmodifiedSincePresent    bool
	IfUnmotModifiedSincePresent bool
	NeedRevalidation            bool
	NotModified                 bool
	IfModifiedSince             time.Time
	IfUnmodifiedSince           time.Time
	IfNoneMatch                 []string
	IfMatch                     []string
	RequestETags                []string
	ResponseETag                string
}

func ParseRequest(req *http.Request) *core.Revalidator {
	// The entity-tags may be spread over several lines as well as comma
	// separated on a single one, so they are flattened before any of them can
	// be compared to a stored tag.
	rqEtags := withQuotingVariants(HeaderAllCommaSepValues(req.Header, "If-None-Match"))
	validator := core.Revalidator{
		NotModified:  len(rqEtags) > 0,
		RequestETags: rqEtags,
	}
	// If-Modified-Since
	if ifModifiedSince := req.Header.Get("If-Modified-Since"); ifModifiedSince != "" {
		validator.IfModifiedSincePresent = true
		validator.IfModifiedSince, _ = ParseHTTPDate(ifModifiedSince)
		validator.NeedRevalidation = true
	}

	// If-Unmodified-Since
	if ifUnmodifiedSince := req.Header.Get("If-Unmodified-Since"); ifUnmodifiedSince != "" {
		validator.IfUnmodifiedSincePresent = true
		validator.IfUnmodifiedSince, _ = ParseHTTPDate(ifUnmodifiedSince)
		validator.NeedRevalidation = true
	}

	// If-None-Match
	if len(rqEtags) > 0 {
		validator.IfNoneMatchPresent = true
		validator.IfNoneMatch = rqEtags
	}

	return &validator
}

// withQuotingVariants adds the quoted twin of every unquoted entity-tag and
// the other way around. Entity-tags are meant to be quoted, but origins do
// send them bare, and a stored tag should still match the one a client
// presents when only the quoting differs.
func withQuotingVariants(etags []string) []string {
	variants := make([]string, 0, len(etags)*2)

	for _, etag := range etags {
		variants = append(variants, etag)

		switch {
		case etag == "" || etag == "*":
		case strings.HasPrefix(etag, `"`) && strings.HasSuffix(etag, `"`) && len(etag) > 1:
			variants = append(variants, strings.Trim(etag, `"`))
		case !strings.HasPrefix(etag, "W/"):
			variants = append(variants, `"`+etag+`"`)
		}
	}

	return variants
}
