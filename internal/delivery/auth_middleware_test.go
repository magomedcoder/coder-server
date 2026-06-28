package delivery

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/magomedcoder/coder-server/internal/config"
)

func TestCheckAPIKeyHeader(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", nil)
	req.Header.Set("X-API-Key", "test-secret")
	if !checkAPIKey(req, "test-secret") {
		t.Fatal("ожидается совпадение X-API-ключа")
	}
}

func TestCheckAPIKeyBearer(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	if !checkAPIKey(req, "test-secret") {
		t.Fatal("ожидается, что токен Bearer будет соответствовать\n")
	}
}

func TestCheckAPIKeyRejectsWrongKey(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", nil)
	req.Header.Set("X-API-Key", "wrong")
	if checkAPIKey(req, "expected") {
		t.Fatal("ожидаемое несоответствие\n")
	}
}

func TestAuthMiddlewareBlocksProtectedPath(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{APIKey: "secret"}
	streams := NewActiveStreamsTracker()
	called := false
	handler := WithMiddleware(cfg, streams, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/chat", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}

	if called {
		t.Fatal("oбработчик не должен запускаться без ключа апи")
	}
}

func TestAuthMiddlewareAllowsPublicHealth(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{APIKey: "secret"}
	streams := NewActiveStreamsTracker()
	called := false
	handler := WithMiddleware(cfg, streams, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}

	if !called {
		t.Fatal("health должно обходить аутентификацию")
	}
}

func TestAuthMiddlewareAllowsWithMatchingKey(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{APIKey: "secret"}
	streams := NewActiveStreamsTracker()
	called := false
	handler := WithMiddleware(cfg, streams, nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/agent/step", nil)
	req.Header.Set("X-API-Key", "secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}

	if !called {
		t.Fatal("handler should run with valid key")
	}
}
