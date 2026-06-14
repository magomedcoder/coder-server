package idempotency

import (
	"sync"
	"time"
)

type idempotencyEntry struct {
	status  int
	body    []byte
	expires time.Time
}

type Store struct {
	mu      sync.RWMutex
	entries map[string]idempotencyEntry
	ttl     time.Duration
}

func New(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	s := &Store{
		entries: make(map[string]idempotencyEntry),
		ttl:     ttl,
	}

	go s.cleanupLoop()

	return s
}

func (s *Store) Get(key string) (status int, body []byte, ok bool) {
	if s == nil || key == "" {
		return 0, nil, false
	}

	s.mu.RLock()
	e, ok := s.entries[key]
	s.mu.RUnlock()

	if !ok || time.Now().After(e.expires) {
		return 0, nil, false
	}

	return e.status, append([]byte(nil), e.body...), true
}

func (s *Store) Put(key string, status int, body []byte) {
	if s == nil || key == "" {
		return
	}

	s.mu.Lock()
	s.entries[key] = idempotencyEntry{
		status:  status,
		body:    append([]byte(nil), body...),
		expires: time.Now().Add(s.ttl),
	}

	s.mu.Unlock()
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

func (s *Store) cleanup() {
	if s == nil {
		return
	}

	now := time.Now()
	s.mu.Lock()
	for k, e := range s.entries {
		if now.After(e.expires) {
			delete(s.entries, k)
		}
	}

	s.mu.Unlock()
}
