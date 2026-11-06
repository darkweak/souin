package rfc

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pierrec/lz4/v4"
)

func compress(t *testing.T, b []byte) []byte {
	t.Helper()

	var out bytes.Buffer
	w := lz4.NewWriter(&out)
	if _, err := w.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	return out.Bytes()
}

func storedResponse(expiry time.Time) []byte {
	return []byte("HTTP/1.1 200 OK\r\nContent-Length: 4\r\nEtag: \"v1\"\r\nX-Souin-Stored-Expiry: " +
		expiry.Format(time.RFC3339Nano) + "\r\n\r\nbody")
}

func decode(t *testing.T, b []byte) *http.Response {
	t.Helper()

	if bytes.HasPrefix(b, lz4Magic) {
		decoded, err := io.ReadAll(lz4.NewReader(bytes.NewReader(b)))
		if err != nil {
			t.Fatal(err)
		}
		b = decoded
	}

	res, err := http.ReadResponse(bufioReader(b), nil)
	if err != nil {
		t.Fatal(err)
	}

	return res
}

func TestSoftPurgeStored(t *testing.T) {
	now := time.Now()
	noRetention := func(http.Header) time.Duration { return 0 }

	for name, compressed := range map[string]bool{"plain": false, "lz4": true} {
		t.Run(name, func(t *testing.T) {
			stored := storedResponse(now.Add(time.Hour))
			if compressed {
				stored = compress(t, stored)
			}

			out, ttl, err := SoftPurgeStored(stored, now, 30*time.Second, noRetention)
			if err != nil {
				t.Fatal(err)
			}
			if compressed != bytes.HasPrefix(out, lz4Magic) {
				t.Errorf("the entry must keep the encoding the storer gave it")
			}

			res := decode(t, out)
			body, _ := io.ReadAll(res.Body)
			if string(body) != "body" || res.Header.Get("Etag") != `"v1"` {
				t.Errorf("the response must be kept as is, got %q %v", body, res.Header)
			}
			if !IsSoftPurged(res.Header) {
				t.Error("the response must be marked")
			}
			if expiry, ok := StoredExpiry(res.Header); !ok || !expiry.Equal(now) || IsStoredFresh(res.Header, now.Add(time.Nanosecond)) {
				t.Errorf("the response must stop being fresh at now, got %v", expiry)
			}
			// It was fresh for another hour: that hour and the stale window
			// are the room it still has.
			if want := time.Hour + 30*time.Second; ttl != want {
				t.Errorf("expected a lifetime of %v, got %v", want, ttl)
			}
		})
	}
}

func TestSoftPurgeStoredAlreadyStale(t *testing.T) {
	now := time.Now()
	retention := func(http.Header) time.Duration { return time.Minute }

	// Stale for 10s out of the 90s it is kept: 80s are left.
	_, ttl, err := SoftPurgeStored(storedResponse(now.Add(-10*time.Second)), now, 30*time.Second, retention)
	if err != nil || ttl != 80*time.Second {
		t.Errorf("expected 80s and no error, got %v and %v", ttl, err)
	}

	_, _, err = SoftPurgeStored(storedResponse(now.Add(-time.Hour)), now, 30*time.Second, retention)
	if !errors.Is(err, ErrSoftPurgeExpired) {
		t.Errorf("an entry past its retention has nothing to purge, got %v", err)
	}
}

func bufioReader(b []byte) *bufio.Reader { return bufio.NewReader(bytes.NewReader(b)) }

func TestSoftPurgeStoredIsIdempotent(t *testing.T) {
	now := time.Now()
	noRetention := func(http.Header) time.Duration { return 0 }

	once, _, err := SoftPurgeStored(storedResponse(now.Add(time.Hour)), now, 0, noRetention)
	if err != nil {
		t.Fatal(err)
	}

	// Measured from the moved expiry the entry would look expired, which it is not.
	if _, _, err := SoftPurgeStored(once, now.Add(time.Minute), 0, noRetention); !errors.Is(err, ErrSoftPurged) {
		t.Errorf("a second soft purge must leave the entry alone, got %v", err)
	}
}
