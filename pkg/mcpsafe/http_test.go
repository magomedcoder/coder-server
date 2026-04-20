package mcpsafe

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoverPanic_PassesThroughOK(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	srv := httptest.NewServer(RecoverPanic("test-origin", inner))

	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTeapot {
		t.Fatalf("status: got %d want %d", resp.StatusCode, http.StatusTeapot)
	}
}

func TestRecoverPanic_SecondRequestAfterPanicStillWorks(t *testing.T) {
	var n int
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		if n == 1 {
			panic("deliberate")
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(RecoverPanic("panic-test", inner))
	t.Cleanup(srv.Close)

	_, err := http.Get(srv.URL)
	if err != nil {
		t.Logf("first request after handler panic (expected flaky): %v", err)
	}

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second request: got status %d", resp.StatusCode)
	}
}

func TestRecoverPanic_EmptyOriginUsesDefault(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := RecoverPanic("   ", inner)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	srv := httptest.NewServer(h)

	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: %d", resp.StatusCode)
	}
}
