package rfc

import (
	"net/http"
	"time"
)

type Revalidator struct {
	Matched                     bool
	IfNoneMatchPresent          bool
	IfMatchPresent              bool
	IfModifiedSincePresent      bool
	IfUnmodifiedSincePresent    bool
	IfUnmotModifiedSincePresent bool
	NeedRevalidation            bool
	IfModifiedSince             time.Time
	IfUnmodifiedSince           time.Time
	IfNoneMatch                 []string
	IfMatch                     []string
}

func ParseRequest(req *http.Request) *Revalidator {
	validator := Revalidator{}
	// If-Modified-Since
	if ifModifiedSince := req.Header.Get("If-Modified-Since"); ifModifiedSince != "" {
		validator.IfModifiedSincePresent = true
		validator.IfModifiedSince, _ = time.Parse(time.RFC1123, ifModifiedSince)
		validator.NeedRevalidation = true
	}

	// If-Unmodified-Since
	if ifUnmodifiedSince := req.Header.Get("If-Unmodified-Since"); ifUnmodifiedSince != "" {
		validator.IfUnmodifiedSincePresent = true
		validator.IfUnmodifiedSince, _ = time.Parse(time.RFC1123, ifUnmodifiedSince)
		validator.NeedRevalidation = true
	}

	// If-None-Match
	if ifNoneMatches := req.Header.Values("If-None-Match"); len(ifNoneMatches) > 0 {
		validator.IfNoneMatchPresent = true
		validator.IfNoneMatch = ifNoneMatches
	}

	return &validator
}

func ValidateETag(res *http.Response, validator *Revalidator) {
	etag := res.Header.Get("ETag")
	validator.Matched = etag == ""

	// If-None-Match
	if validator.IfNoneMatchPresent {
		for _, ifNoneMatch := range validator.IfNoneMatch {
			// Asrterisk special char to match any of ETag
			if ifNoneMatch == "*" {
				validator.Matched = false
				return
			}
			if ifNoneMatch == etag {
				validator.Matched = false
				return
			}
		}

		validator.Matched = true
		return
	}

	// If-Match
	if validator.IfMatchPresent {
		validator.Matched = false
		if etag == "" {
			return
		}

		for _, ifMatch := range validator.IfMatch {
			// Asrterisk special char to match any of ETag
			if ifMatch == "*" {
				validator.Matched = true
				return
			}
			if ifMatch == etag {
				validator.Matched = true
				return
			}
		}
	}
}
