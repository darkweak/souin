package rfc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/darkweak/souin/pkg/storage/types"
)

func ValidateETagFromHeader(etag string, validator *types.Revalidator) {
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

func ParseRequest(req *http.Request) *types.Revalidator {
	var rqEtags []string
	if len(req.Header.Get("If-None-Match")) > 0 {
		rqEtags = strings.Split(req.Header.Get("If-None-Match"), ",")
	}
	for i, tag := range rqEtags {
		rqEtags[i] = strings.Trim(tag, " ")
	}
	validator := types.Revalidator{
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

func DecodeMapping(item []byte) (*StorageMapper, error) {
	mapping := &StorageMapper{}
	e := json.Unmarshal(item, mapping)

	return mapping, e
}

func MappingElection(provider types.Storer, item []byte, req *http.Request, validator *types.Revalidator) (resultFresh *http.Response, resultStale *http.Response, e error) {
	mapping := &StorageMapper{}

	if len(item) != 0 {
		mapping, e = DecodeMapping(item)
		if e != nil {
			return resultFresh, resultStale, e
		}
	}

	for keyName, keyItem := range mapping.Mapping {
		valid := true

		for hname, hval := range keyItem.VariedHeaders {
			if req.Header.Get(hname) != strings.Join(hval, ", ") {
				valid = false

				break
			}
		}

		if !valid {
			continue
		}

		ValidateETagFromHeader(keyItem.Etag, validator)

		if validator.Matched {
			// If the key is fresh enough.
			if time.Since(keyItem.FreshTime) < 0 {
				response := provider.Get(keyName)
				if response != nil {
					if resultFresh, e = http.ReadResponse(bufio.NewReader(bytes.NewBuffer(response)), req); e != nil {
						return resultFresh, resultStale, e
					}

					return resultFresh, resultStale, e
				}
			}

			// If the key is still stale.
			if time.Since(keyItem.StaleTime) < 0 {
				response := provider.Get(keyName)
				if response != nil {
					if resultStale, e = http.ReadResponse(bufio.NewReader(bytes.NewBuffer(response)), req); e != nil {
						return resultFresh, resultStale, e
					}
				}
			}
		}
	}

	return resultFresh, resultStale, e
}

type KeyIndex struct {
	StoredAt      time.Time           `json:"stored_at,omitempty"`
	FreshTime     time.Time           `json:"fresh_time,omitempty"`
	StaleTime     time.Time           `json:"stale_time,omitempty"`
	VariedHeaders map[string][]string `json:"varied_headers,omitempty"`
	Etag          string              `json:"etag,omitempty"`
	RealKey       string              `json:"real_key,omitempty"`
}
type StorageMapper struct {
	Mapping map[string]*KeyIndex `json:"mapping,omitempty"`
}

func MappingUpdater(key string, item []byte, now, freshTime, staleTime time.Time, variedHeaders http.Header, etag, realKey string) (val []byte, e error) {
	mapping := &StorageMapper{}
	if len(item) != 0 {
		e = json.Unmarshal(item, mapping)
		if e != nil {
			return nil, e
		}
	}

	if mapping.Mapping == nil {
		mapping.Mapping = make(map[string]*KeyIndex)
	}

	var pbvariedeheader map[string][]string
	if variedHeaders != nil {
		pbvariedeheader = make(map[string][]string)
	}

	for k, v := range variedHeaders {
		pbvariedeheader[k] = append(pbvariedeheader[k], v...)
	}

	mapping.Mapping[key] = &KeyIndex{
		StoredAt:      now,
		FreshTime:     freshTime,
		StaleTime:     staleTime,
		VariedHeaders: pbvariedeheader,
		Etag:          etag,
		RealKey:       realKey,
	}

	val, e = json.Marshal(mapping)
	if e != nil {
		return nil, e
	}

	return val, e
}
