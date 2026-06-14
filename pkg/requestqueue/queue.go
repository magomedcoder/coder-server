package requestqueue

import (
	"context"
	"errors"
	"time"
)

var ErrQueueTimeout = errors.New("очередь переполнена: превышено время ожидания слота")

type Queue struct {
	sem     chan struct{}
	maxWait time.Duration
}

func New(maxConcurrent int, maxWait time.Duration) *Queue {
	if maxConcurrent <= 0 {
		maxConcurrent = 8
	}

	return &Queue{
		sem:     make(chan struct{}, maxConcurrent),
		maxWait: maxWait,
	}
}

func (q *Queue) Acquire(ctx context.Context) error {
	if q == nil {
		return nil
	}

	waitCtx := ctx
	var cancel context.CancelFunc
	if q.maxWait > 0 {
		waitCtx, cancel = context.WithTimeout(ctx, q.maxWait)
		defer cancel()
	}

	select {
	case q.sem <- struct{}{}:
		return nil
	case <-waitCtx.Done():
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return ErrQueueTimeout
		}
		return waitCtx.Err()
	}
}

func (q *Queue) Release() {
	if q == nil {
		return
	}

	select {
	case <-q.sem:
	default:
	}
}

func (q *Queue) InFlight() int {
	if q == nil {
		return 0
	}

	return len(q.sem)
}

func (q *Queue) Capacity() int {
	if q == nil {
		return 0
	}

	return cap(q.sem)
}
