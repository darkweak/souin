package scep

// Logger is the fundamental interface for all SCEP logging operations. It
// has the same signature as the `github.com/go-kit/kit/log` interface, to
// allow for interop between the two. Log creates a log event from keyvals,
// a variadic sequence of alternating keys and values. Implementations must
// be safe for concurrent use by multiple goroutines. In particular, any
// implementation of Logger that appends to keyvals or modifies or retains
// any of its elements must make a copy first.
type Logger interface {
	Log(keyvals ...interface{}) error
}

type nopLogger struct{}

// newNopLogger returns a logger that logs nothing.
func newNopLogger() Logger { return nopLogger{} }

func (nopLogger) Log(...interface{}) error { return nil }
