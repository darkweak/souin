package core

// Yaegi compatible replacement of the protobuf generated storage.pb.go.
// The protobuf runtime crashes the Yaegi compiler (even with unsafe enabled),
// so the messages are plain structs encoded as JSON.
// The exported names and getters mirror the generated code so the vendored
// Souin sources compile against both implementations.

import (
	"time"
)

// Timestamp mirrors the google.protobuf.Timestamp message.
type Timestamp struct {
	Seconds int64 `json:"seconds,omitempty"`
	Nanos   int32 `json:"nanos,omitempty"`
}

// NewTimestamp mirrors timestamppb.New.
func NewTimestamp(t time.Time) *Timestamp {
	return &Timestamp{Seconds: t.Unix(), Nanos: int32(t.Nanosecond())}
}

// AsTime mirrors timestamppb.Timestamp.AsTime.
func (x *Timestamp) AsTime() time.Time {
	if x == nil {
		return time.Unix(0, 0).UTC()
	}

	return time.Unix(x.Seconds, int64(x.Nanos)).UTC()
}

type KeyIndex struct {
	StoredAt      *Timestamp                     `json:"stored_at,omitempty"`
	FreshTime     *Timestamp                     `json:"fresh_time,omitempty"`
	StaleTime     *Timestamp                     `json:"stale_time,omitempty"`
	VariedHeaders map[string]*KeyIndexStringList `json:"varied_headers,omitempty"`
	Etag          string                         `json:"etag,omitempty"`
	RealKey       string                         `json:"real_key,omitempty"`
}

func (x *KeyIndex) GetStoredAt() *Timestamp {
	if x != nil {
		return x.StoredAt
	}

	return nil
}

func (x *KeyIndex) GetFreshTime() *Timestamp {
	if x != nil {
		return x.FreshTime
	}

	return nil
}

func (x *KeyIndex) GetStaleTime() *Timestamp {
	if x != nil {
		return x.StaleTime
	}

	return nil
}

func (x *KeyIndex) GetVariedHeaders() map[string]*KeyIndexStringList {
	if x != nil {
		return x.VariedHeaders
	}

	return nil
}

func (x *KeyIndex) GetEtag() string {
	if x != nil {
		return x.Etag
	}

	return ""
}

func (x *KeyIndex) GetRealKey() string {
	if x != nil {
		return x.RealKey
	}

	return ""
}

type StorageMapper struct {
	Mapping map[string]*KeyIndex `json:"mapping,omitempty"`
}

func (x *StorageMapper) GetMapping() map[string]*KeyIndex {
	if x != nil {
		return x.Mapping
	}

	return nil
}

type KeyIndexStringList struct {
	HeaderValue []string `json:"header_value,omitempty"`
}

func (x *KeyIndexStringList) GetHeaderValue() []string {
	if x != nil {
		return x.HeaderValue
	}

	return nil
}
