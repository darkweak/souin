package api

import (
	"strings"
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/storage"
	"github.com/darkweak/souin/tests"
	"github.com/darkweak/storages/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEvictMapping_OversizedValue(t *testing.T) {
	memoryStorer, err := storage.Factory(tests.MockConfiguration(tests.BaseConfiguration))
	if err != nil {
		t.Fatalf("Impossible to create the storer, %v given", err)
	}

	oversized := strings.Repeat("a", maxMappingValueSize+1)
	if err := memoryStorer.Set(core.MappingKeyPrefix+"oversized", []byte(oversized), time.Minute); err != nil {
		t.Fatalf("Impossible to store the value, %v given", err)
	}

	expired, _ := proto.Marshal(&core.StorageMapper{Mapping: map[string]*core.KeyIndex{
		"expired-key": {
			FreshTime: timestamppb.New(time.Now().Add(-time.Hour)),
			StaleTime: timestamppb.New(time.Now().Add(-time.Hour)),
			RealKey:   "expired-key",
		},
	}})
	if err := memoryStorer.Set(core.MappingKeyPrefix+"valid", expired, time.Minute); err != nil {
		t.Fatalf("Impossible to store the value, %v given", err)
	}

	EvictMapping(memoryStorer)

	if res := memoryStorer.Get(core.MappingKeyPrefix + "oversized"); len(res) != 0 {
		t.Error("The oversized mapping should be deleted without being decoded")
	}

	if res := memoryStorer.Get(core.MappingKeyPrefix + "valid"); len(res) != 0 {
		t.Error("The fully expired mapping should be deleted")
	}
}
