package core

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pierrec/lz4/v4"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MaxMappingSize bounds the legacy mapping blobs a storer is willing to
// decode or migrate; larger values are dropped instead of materialized.
const MaxMappingSize = 1 << 20

// Lz4WriterPool pools lz4 writers, which are safe to reuse once Close has
// flushed the frame. Callers must Reset the writer before use and only
// return it to the pool after Close. Readers must never be pooled this way:
// a pooled reader escapes through http.Response.Body and would be recycled
// while another goroutine still reads from it.
var Lz4WriterPool = sync.Pool{New: func() any { return lz4.NewWriter(nil) }}

// MappingWalker is an optional interface a Storer can implement to stream
// mapping entries in bounded batches instead of materializing the whole
// mapping index in memory like MapKeys does. The walk stops early when fn
// returns false. The key passed to fn is stripped of the given prefix.
type MappingWalker interface {
	WalkMappings(prefix string, fn func(key string, value []byte) bool) error
}

// SetStorer is an optional interface a Storer can implement to store a set
// of members under a key natively, instead of a separator-joined string
// value that must be fully read and rewritten on every addition.
type SetStorer interface {
	// AddToSet adds members to the set stored at key, extending the set
	// lifetime to at least the given duration when it is positive, without
	// shortening a longer remaining lifetime.
	AddToSet(key string, members []string, duration time.Duration) error
	// GetSet returns all members of the set stored at key.
	GetSet(key string) []string
	// WalkSets visits every set whose key matches the prefix. The walk stops
	// early when fn returns false. The key passed to fn is stripped of the
	// given prefix.
	WalkSets(prefix string, fn func(key string, members []string) bool) error
}

// MappingEntry marshals a single mapping entry for storers that keep one
// value per varied key instead of one blob per base key.
func MappingEntry(now, freshTime, staleTime time.Time, variedHeaders http.Header, etag, realKey string) ([]byte, error) {
	var pbvariedeheader map[string]*KeyIndexStringList
	if variedHeaders != nil {
		pbvariedeheader = make(map[string]*KeyIndexStringList, len(variedHeaders))
	}

	for headerName, headerValues := range variedHeaders {
		pbvariedeheader[headerName] = &KeyIndexStringList{HeaderValue: headerValues}
	}

	return proto.Marshal(&KeyIndex{
		StoredAt:      timestamppb.New(now),
		FreshTime:     timestamppb.New(freshTime),
		StaleTime:     timestamppb.New(staleTime),
		VariedHeaders: pbvariedeheader,
		Etag:          etag,
		RealKey:       realKey,
	})
}

var registered = sync.Map{}

func RegisterStorage(s Storer) {
	_ = s.Init()
	registered.Store(fmt.Sprintf("%s-%s", s.Name(), s.Uuid()), s)
}

func GetRegisteredStorer(name string) Storer {
	s, _ := registered.Load(name)
	if s != nil {
		if v, ok := s.(Storer); ok {
			return v
		}
	}

	return nil
}

func ResetRegisteredStorages() {
	registered.Range(func(key, value any) bool {
		registered.Delete(key)

		return true
	})

	registered = sync.Map{}
}

func GetRegisteredStorers() []Storer {
	storers := make([]Storer, 0)

	registered.Range(func(_, value any) bool {
		if s, ok := value.(Storer); ok {
			storers = append(storers, s)
		}

		return true
	})

	return storers
}
