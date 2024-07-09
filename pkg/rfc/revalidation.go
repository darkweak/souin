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
	var rqEtags []string
	if len(req.Header.Get("If-None-Match")) > 0 {
		rqEtags = strings.Split(req.Header.Get("If-None-Match"), ",")
	}
	for i, tag := range rqEtags {
		rqEtags[i] = strings.Trim(tag, " ")
	}
	validator := core.Revalidator{
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
