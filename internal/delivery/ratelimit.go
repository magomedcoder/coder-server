package delivery

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/magomedcoder/coder-server/internal/domain"
)

type rateLimiter struct {
	mu      sync.Mutex
	limit   float64
	burst   float64
	clients map[string]*rateClient
}

type rateClient struct {
	tokens float64
	last   time.Time
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	if requestsPerMinute <= 0 {
		return nil
	}

	return &rateLimiter{
		limit:   float64(requestsPerMinute) / 60.0,
		burst:   float64(requestsPerMinute),
		clients: make(map[string]*rateClient),
	}
}

func (rl *rateLimiter) allow(key string) bool {
	if rl == nil {
		return true
	}

	now := time.Now()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, ok := rl.clients[key]
	if !ok {
		rl.clients[key] = &rateClient{
			tokens: rl.burst - 1,
			last:   now,
		}
		return true
	}

	elapsed := now.Sub(c.last).Seconds()
	c.tokens += elapsed * rl.limit
	if c.tokens > rl.burst {
		c.tokens = rl.burst
	}
	c.last = now

	if c.tokens < 1 {
		return false
	}
	c.tokens--
	return true
}

func clientRateLimitKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func WithRateLimit(requestsPerMinute int, next http.Handler) http.Handler {
	rl := newRateLimiter(requestsPerMinute)
	if rl == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		key := clientRateLimitKey(r)
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			key = "key:" + apiKey
		}

		if !rl.allow(key) {
			writeJSON(w, http.StatusTooManyRequests, domain.NewErrorResponse("rate_limit_exceeded", "превышен лимит запросов"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
