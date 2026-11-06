package rfc

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/pierrec/lz4/v4"
	"github.com/pquerna/cachecontrol/cacheobject"
)

// lz4Magic opens every frame the storers compress a response into.
var lz4Magic = []byte{0x04, 0x22, 0x4D, 0x18}

// SoftPurgedHeader marks a stored response a soft purge made stale. It lives
// in the stored entry next to the stored expiry, so the mark costs no extra
// storage key nor read, and it disappears with the entry as soon as a refresh
// replaces it.
const SoftPurgedHeader = "X-Souin-Soft-Purged"

// RetentionWindow returns for how long a response is kept once it turned
// stale: the extra room a `stale-while-revalidate` asks for. The storer adds
// the configured stale window on its own.
func RetentionWindow(responseCc *cacheobject.ResponseCacheDirectives) time.Duration {
	if responseCc == nil {
		return 0
	}

	return max(time.Duration(responseCc.StaleWhileRevalidate)*time.Second, 0)
}

// ErrSoftPurgeExpired is returned when the entry reached the end of its
// retention already: there is nothing left to soft purge.
var ErrSoftPurgeExpired = errors.New("the stored response is already expired")

// ErrSoftPurged is returned when the entry was soft purged already. Purging it
// again would change nothing but the lifetime: that one is measured from the
// stored expiry, which the first soft purge moved.
var ErrSoftPurged = errors.New("the stored response is already soft purged")

// IsSoftPurged tells whether the stored response was made stale by a soft purge.
func IsSoftPurged(h http.Header) bool {
	return h.Get(SoftPurgedHeader) != ""
}

// SoftPurgeStored rewrites a stored response so it stops being fresh at now.
// The existing stale path then serves and refreshes it like any other stale
// response, which is the whole soft purge.
//
// It returns the rewritten entry and how long the storer has to keep it. That
// lifetime is the one the entry still had: freshness ends sooner, retention
// does not move. retention is the extra room a response asks to stay
// available once stale, and stale the configured stale window.
func SoftPurgeStored(
	stored []byte,
	now time.Time,
	stale time.Duration,
	retention func(http.Header) time.Duration,
) ([]byte, time.Duration, error) {
	// The storers keep the response compressed, and the raw write the caller
	// does afterwards is not going to compress it for them.
	compressed := bytes.HasPrefix(stored, lz4Magic)
	plain := stored
	if compressed {
		var err error
		if plain, err = io.ReadAll(lz4.NewReader(bytes.NewReader(stored))); err != nil {
			return nil, 0, err
		}
	}

	res, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(plain)), nil)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}

	if IsSoftPurged(res.Header) {
		return nil, 0, ErrSoftPurged
	}

	// The lifetime is measured from the instant the response stopped being
	// fresh. An entry without a stored expiry cannot tell, so it gets the
	// longest it could have been given from now.
	remaining := retention(res.Header) + stale
	if expiry, ok := StoredExpiry(res.Header); ok {
		remaining += expiry.Sub(now)
	}
	if remaining <= 0 {
		return nil, 0, ErrSoftPurgeExpired
	}

	// Content-Length is the framing http.ReadResponse consumed; the stored
	// one is the original and stays authoritative.
	res.Header.Set(SoftPurgedHeader, now.Format(time.RFC3339Nano))
	SetStoredExpiry(res.Header, now)

	var buf bytes.Buffer
	buf.Grow(len(plain) + 64)
	buf.WriteString("HTTP/1.1 ")
	buf.WriteString(strconv.Itoa(res.StatusCode))
	buf.WriteString(" ")
	buf.WriteString(http.StatusText(res.StatusCode))
	buf.WriteString("\r\n")
	if err := res.Header.Write(&buf); err != nil {
		return nil, 0, err
	}
	buf.WriteString("\r\n")
	buf.Write(body)

	if !compressed {
		return buf.Bytes(), remaining, nil
	}

	var out bytes.Buffer
	writer := lz4.NewWriter(&out)
	if _, err := writer.Write(buf.Bytes()); err != nil {
		return nil, 0, err
	}
	if err := writer.Close(); err != nil {
		return nil, 0, err
	}

	return out.Bytes(), remaining, nil
}
