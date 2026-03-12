package core

import (
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
	validator.NeedRevalidation = validator.NeedRevalidation || (validator.ResponseETag != "" && len(validator.RequestETags) > 0)
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
