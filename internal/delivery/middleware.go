package delivery

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/service"
)

type requestIDKey struct{}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

func WithMiddleware(cfg *config.Config, streams *ActiveStreams, metrics *service.Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if cfg != nil && cfg.AuthEnabled() && !isPublicPath(r.URL.Path) {
			if !checkAPIKey(r, cfg.APIKey) {
				writeJSON(w, http.StatusUnauthorized, domain.NewErrorResponse("unauthorized", "неверный или отсутствующий API key"))
				return
			}
		}

		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", requestID)

		ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
		start := time.Now()

		next.ServeHTTP(w, r.WithContext(ctx))

		durationMs := time.Since(start).Milliseconds()
		if metrics != nil {
			metrics.RecordRequest(durationMs)
		}
		if cfg != nil && cfg.Logging.Structured {
			log.Printf(`{"request_id":%q,"method":%q,"path":%q,"duration_ms":%d,"active_streams":%d}`,
				requestID, r.Method, r.URL.Path, durationMs, streams.Count())
		} else {
			log.Printf("request_id=%s method=%s path=%s duration_ms=%d active_streams=%d",
				requestID, r.Method, r.URL.Path, durationMs, streams.Count())
		}
	})
}

func isPublicPath(path string) bool {
	switch path {
	case "/v1/health", "/v1/health/live", "/v1/health/ready", "/v1/metrics":
		return true
	default:
		return false
	}
}

func checkAPIKey(r *http.Request, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}

	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key == expected
	}

	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:]) == expected
	}

	return false
}

func resolveRequestID(ctx context.Context, bodyID *string) string {
	if bodyID != nil {
		if id := strings.TrimSpace(*bodyID); id != "" {
			return id
		}
	}
	return RequestIDFromContext(ctx)
}

func mapRunnerError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrRunnerModelNotLoaded()) {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", "модель не загружена на gen-runner"))
		return
	}
	if errors.Is(err, service.ErrQueueTimeout) {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("overloaded", "сервер перегружен, повторите позже"))
		return
	}

	msg := err.Error()
	if strings.Contains(msg, "модель не загружена") {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", msg))
		return
	}
	if strings.Contains(msg, "gen-runner") || strings.Contains(msg, "раннер") {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", "gen-runner недоступен"))
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.NewErrorResponse("internal_error", "ошибка генерации"))
}

func ensureRunnerReady(ctx context.Context, llm *service.LLMRunnerService, w http.ResponseWriter) bool {
	ok, err := llm.CheckConnection(ctx)
	if err != nil || !ok {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", "gen-runner недоступен"))
		return false
	}

	if err := llm.ModelReady(ctx); err != nil {
		if errors.Is(err, domain.ErrRunnerModelNotLoaded()) {
			writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", "модель не загружена на gen-runner"))
			return false
		}
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", err.Error()))
		return false
	}

	return true
}

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Request-ID, Last-Event-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
