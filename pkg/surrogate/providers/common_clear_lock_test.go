package providers

import (
	"sync"
	"testing"
	"time"

	"github.com/darkweak/souin/pkg/storage/types"
)

// gateStorer holds storeTag inside its read of one tag until the test releases
// it, and reports when DeleteMany runs. Embedding only types.Storer hides any
// SetStorer methods of the wrapped storer, so storeTag takes its
// read-modify-write path.
type gateStorer struct {
	types.Storer
	gatedKey  string
	readOnce  sync.Once
	readTag   chan struct{}
	release   chan struct{}
	delOnce   sync.Once
	deleteRan chan struct{}
}

func (g *gateStorer) Get(key string) []byte {
	if key == g.gatedKey {
		g.readOnce.Do(func() {
			close(g.readTag)
			<-g.release
		})
	}

	return g.Storer.Get(key)
}

func (g *gateStorer) DeleteMany(key string) {
	g.delOnce.Do(func() { close(g.deleteRan) })
	g.Storer.DeleteMany(key)
}

// Clear must not interleave with a storeTag that is between reading and
// rewriting a tag. Without the lock, Clear deletes the tag in between and the
// rewrite brings it back, so a tag stored before the flush survives it.
func TestBaseStorage_Clear_WaitsForAnInFlightStoreTag(t *testing.T) {
	bs := mockCommonProvider()
	bs.duration = time.Hour

	g := &gateStorer{
		Storer:    bs.Storage,
		gatedKey:  surrogatePrefix + "tagA",
		readTag:   make(chan struct{}),
		release:   make(chan struct{}),
		deleteRan: make(chan struct{}),
	}
	bs.Storage = g

	storeDone := make(chan struct{})
	go func() {
		bs.storeTag("tagA", "k1")
		close(storeDone)
	}()
	<-g.readTag

	clearDone := make(chan struct{})
	go func() {
		bs.Clear()
		close(clearDone)
	}()

	// Give an unlocked Clear the chance to run while storeTag is mid-update.
	select {
	case <-g.deleteRan:
	case <-time.After(200 * time.Millisecond):
	}

	close(g.release)
	<-storeDone
	<-clearDone

	if got := bs.List(); len(got) != 0 {
		t.Errorf("Clear should wait for the in-flight storeTag and then delete its tag, %v left", got)
	}
}
