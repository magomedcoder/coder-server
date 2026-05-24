package service

import (
	"sync"
	"time"
)

type circuitState struct {
	failures    int
	openUntil   time.Time
	lastFailure time.Time
}

type CircuitBreaker struct {
	mu          sync.Mutex
	states      map[string]*circuitState
	maxFailures int
	cooldown    time.Duration
}

func NewCircuitBreaker(maxFailures int, cooldown time.Duration) *CircuitBreaker {
	if maxFailures <= 0 {
		maxFailures = 3
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		states:      make(map[string]*circuitState),
		maxFailures: maxFailures,
		cooldown:    cooldown,
	}
}

func (cb *CircuitBreaker) Allow(addr string) bool {
	if cb == nil || addr == "" {
		return true
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	st := cb.states[addr]
	if st == nil {
		return true
	}
	if st.openUntil.IsZero() {
		return true
	}
	if time.Now().After(st.openUntil) {
		st.openUntil = time.Time{}
		st.failures = 0
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess(addr string) {
	if cb == nil || addr == "" {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	st, ok := cb.states[addr]
	if !ok {
		return
	}
	st.failures = 0
	st.openUntil = time.Time{}
}

func (cb *CircuitBreaker) RecordFailure(addr string) {
	if cb == nil || addr == "" {
		return
	}

	cb.mu.Lock()
	defer cb.mu.Unlock()

	st, ok := cb.states[addr]
	if !ok {
		st = &circuitState{}
		cb.states[addr] = st
	}

	st.failures++
	st.lastFailure = time.Now()
	if st.failures >= cb.maxFailures {
		st.openUntil = time.Now().Add(cb.cooldown)
	}
}
