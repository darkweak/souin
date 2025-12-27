package middleware

import (
	"context"
	"time"
)

type contextSWR struct {
	ctx context.Context
}

func newContextSWR(ctx context.Context) context.Context {
	return &contextSWR{ctx: ctx}
}

func (*contextSWR) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*contextSWR) Done() <-chan struct{}       { return nil }
func (*contextSWR) Err() error                  { return nil }

func (l *contextSWR) Value(key interface{}) interface{} {
	return l.ctx.Value(key)
}
