package service

import (
	"context"
	"errors"
	"time"
)

var ErrQueueTimeout = errors.New("очередь переполнена: превышено время ожидания слота")

type RequestQueue struct {
	sem     chan struct{}
	maxWait time.Duration
}

func NewRequestQueue(maxConcurrent int, maxWait time.Duration) *RequestQueue {
	if maxConcurrent <= 0 {
		maxConcurrent = 8
	}

	return &RequestQueue{
		sem:     make(chan struct{}, maxConcurrent),
		maxWait: maxWait,
	}
}

func (q *RequestQueue) Acquire(ctx context.Context) error {
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

func (q *RequestQueue) Release() {
	if q == nil {
		return
	}

	select {
	case <-q.sem:
	default:
	}
}

func (q *RequestQueue) InFlight() int {
	if q == nil {
		return 0
	}

	return len(q.sem)
}

func (q *RequestQueue) Capacity() int {
	if q == nil {
		return 0
	}

	return cap(q.sem)
}
