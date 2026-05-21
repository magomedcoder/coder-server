package mcpclient

import (
	"context"
	"errors"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string {
	return "timeout"
}

func (timeoutErr) Timeout() bool {
	return true
}

func (timeoutErr) Temporary() bool {
	return true
}

func TestShouldRetryPooledSessionError(t *testing.T) {
	if shouldRetryPooledSessionError(context.Background(), nil) {
		t.Fatal("nil error не должно быть retried")
	}

	if shouldRetryPooledSessionError(context.Background(), context.Canceled) {
		t.Fatal("context canceled не должно быть retried")
	}

	if !shouldRetryPooledSessionError(context.Background(), context.DeadlineExceeded) {
		t.Fatal("deadline exceeded должно быть retried once for pooled sessions")
	}

	if !shouldRetryPooledSessionError(context.Background(), timeoutErr{}) {
		t.Fatal("net timeout error должно быть retried")
	}

	if !shouldRetryPooledSessionError(context.Background(), errors.New("read: connection reset by peer")) {
		t.Fatal("connection reset должно быть retried")
	}

	if shouldRetryPooledSessionError(context.Background(), errors.New("validation не удалось")) {
		t.Fatal("business errors не должно быть retried")
	}
}

func TestShouldRetryPooledSessionErrorParentContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if shouldRetryPooledSessionError(ctx, errors.New("connection reset by peer")) {
		t.Fatal("не должно retry when parent context is already done")
	}
}
