package providers

import (
	"testing"
	"time"
)

func TestBaseStorage_Clear(t *testing.T) {
	bs := mockCommonProvider()
	bs.duration = time.Hour

	bs.storeTag("tagA", "k1")
	bs.storeTag("tagB", "k2")

	const unrelatedKey = "GET-https-example.com-/SURROGATE_not-a-tag"
	if err := bs.Storage.Set(unrelatedKey, []byte("body"), time.Minute); err != nil {
		t.Fatalf("Impossible to store the unrelated key, %v given", err)
	}

	if len(bs.List()) != 2 {
		t.Fatalf("Both tags should be stored before Clear, %v given", bs.List())
	}

	bs.Clear()

	if got := bs.List(); len(got) != 0 {
		t.Errorf("Clear should delete every surrogate key, %v left", got)
	}

	if got := bs.Storage.Get(unrelatedKey); string(got) != "body" {
		t.Errorf("Clear should only delete keys starting with %s, the unrelated key became %q", surrogatePrefix, got)
	}

	bs.storeTag("tagA", "k3")

	if len(bs.List()) != 1 {
		t.Errorf("The storer should keep storing tags after Clear, %v given", bs.List())
	}
}
