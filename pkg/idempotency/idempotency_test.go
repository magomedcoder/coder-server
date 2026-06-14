package idempotency

import (
	"testing"
	"time"
)

func TestIdempotencyStorePutGet(t *testing.T) {
	store := New(time.Minute)
	store.Put("req-1", 200, []byte(`{"ok":true}`))

	status, body, ok := store.Get("req-1")
	if !ok || status != 200 || string(body) != `{"ok":true}` {
		t.Fatalf("неожиданный Get: ok=%v status=%d body=%q", ok, status, body)
	}

	if _, _, ok := store.Get("missing"); ok {
		t.Fatal("ожидался промах по неизвестному ключу")
	}
}
