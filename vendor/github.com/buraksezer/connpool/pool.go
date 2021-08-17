// Package pool implements a pool of net.Conn interfaces to manage and reuse them.
package connpool

import (
	"context"
	"errors"
	"net"
)

var (
	// ErrClosed is the error resulting if the pool is closed via pool.Close().
	ErrClosed = errors.New("pool is closed")
)

// Pool interface describes a pool implementation. A pool should have maximum
// capacity. An ideal pool is thread-safe and easy to use.
type Pool interface {
	// Get returns a new connection from the pool. Closing the connections puts
	// it back to the Pool. Closing it when the pool is destroyed or full will
	// be counted as an error.
	Get(context.Context) (net.Conn, error)

	// Close closes the pool and all its connections. After Close() the pool is
	// no longer usable.
	Close()

	// Len returns the current number of idle connections of the pool.
	Len() int

	// NumberOfConns returns the total number of alive connections of the pool.
	NumberOfConns() int
}
