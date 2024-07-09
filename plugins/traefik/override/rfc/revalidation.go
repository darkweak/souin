package rfc

import (
	"net/http"
	"strings"
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
	NotModified                 bool
	IfModifiedSince             time.Time
	IfUnmodifiedSince           time.Time
	IfNoneMatch                 []string
	IfMatch                     []string
	RequestETags                []string
	ResponseETag                string
}

func ValidateETagFromHeader(etag string, validator *Revalidator) {
	validator.ResponseETag = etag
	validator.NeedRevalidation = validator.NeedRevalidation || validator.ResponseETag != ""
	validator.Matched = validator.ResponseETag == "" || (validator.ResponseETag != "" && len(validator.RequestETags) == 0)

	if len(validator.RequestETags) == 0 {
		validator.NotModified = false
		return
	}

	// If-None-Match
	if validator.IfNoneMatchPresent {
		for _, ifNoneMatch := range validator.IfNoneMatch {
			// Asrterisk special char to match any of ETag
			if ifNoneMatch == "*" {
				validator.Matched = true
				return
			}
			if ifNoneMatch == validator.ResponseETag {
				validator.Matched = true
				return
			}
		}

		validator.Matched = false
		return
	}

	// If-Match
	if validator.IfMatchPresent {
		validator.Matched = false
		if validator.ResponseETag == "" {
			return
		}

		for _, ifMatch := range validator.IfMatch {
			// Asrterisk special char to match any of ETag
			if ifMatch == "*" {
				validator.Matched = true
				return
			}
			if ifMatch == validator.ResponseETag {
				validator.Matched = true
				return
			}
		}
	}
}

func ParseRequest(req *http.Request) *Revalidator {
	var rqEtags []string
	if len(req.Header.Get("If-None-Match")) > 0 {
		rqEtags = strings.Split(req.Header.Get("If-None-Match"), ",")
	}
	for i, tag := range rqEtags {
		rqEtags[i] = strings.Trim(tag, " ")
	}
	validator := Revalidator{
		NotModified:  len(rqEtags) > 0,
		RequestETags: rqEtags,
	}
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
