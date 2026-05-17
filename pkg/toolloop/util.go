package toolloop

import (
	"context"
	"time"
)

func ToolExecutionDuration(sessionTimeoutSec int32) time.Duration {
	s := int64(sessionTimeoutSec)
	if s <= 0 {
		s = DefaultToolExecSeconds
	}

	if s < MinToolExecSeconds {
		s = MinToolExecSeconds
	}

	if s > MaxToolExecSeconds {
		s = MaxToolExecSeconds
	}

	return time.Duration(s) * time.Second
}

func RunFnWithContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return fn()
	}

	type result struct {
		val T
		err error
	}

	ch := make(chan result, 1)
	go func() {
		v, err := fn()
		ch <- result{v, err}
	}()

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case r := <-ch:
		return r.val, r.err
	}
}
