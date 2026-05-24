package service

import (
	"sync"
	"time"
)

type SSEEvent struct {
	ID    int
	Event string
	Data  string
}

type StreamSession struct {
	requestID string
	events    []SSEEvent
	done      bool
	subs      map[int]chan SSEEvent
	nextSubID int
	mu        sync.RWMutex
}

type StreamRegistry struct {
	mu       sync.RWMutex
	sessions map[string]*StreamSession
	ttl      time.Duration
}

func NewStreamRegistry(ttl time.Duration) *StreamRegistry {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	r := &StreamRegistry{
		sessions: make(map[string]*StreamSession),
		ttl:      ttl,
	}
	go r.cleanupLoop()
	return r
}

func (r *StreamRegistry) Start(requestID string) *StreamSession {
	if r == nil || requestID == "" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.sessions[requestID]; ok {
		existing.mu.Lock()
		existing.done = false
		existing.mu.Unlock()
		return existing
	}

	s := &StreamSession{
		requestID: requestID,
		subs:      make(map[int]chan SSEEvent),
	}
	r.sessions[requestID] = s
	return s
}

func (r *StreamRegistry) Get(requestID string) (*StreamSession, bool) {
	if r == nil || requestID == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sessions[requestID]
	return s, ok
}

func (s *StreamSession) Append(event SSEEvent) {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.events = append(s.events, event)
	subs := make([]chan SSEEvent, 0, len(s.subs))
	for _, ch := range s.subs {
		subs = append(subs, ch)
	}
	s.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *StreamSession) MarkDone() {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.done = true
	for id, ch := range s.subs {
		close(ch)
		delete(s.subs, id)
	}
	s.mu.Unlock()
}

func (s *StreamSession) IsDone() bool {
	if s == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.done
}

func (s *StreamSession) EventsAfter(lastEventID int) []SSEEvent {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]SSEEvent, 0)
	for _, e := range s.events {
		if e.ID > lastEventID {
			out = append(out, e)
		}
	}
	return out
}

func (s *StreamSession) Subscribe() (<-chan SSEEvent, func()) {
	if s == nil {
		ch := make(chan SSEEvent)
		close(ch)
		return ch, func() {}
	}

	s.mu.Lock()
	id := s.nextSubID
	s.nextSubID++
	ch := make(chan SSEEvent, 64)
	s.subs[id] = ch
	s.mu.Unlock()

	unsub := func() {
		s.mu.Lock()
		if c, ok := s.subs[id]; ok {
			delete(s.subs, id)
			close(c)
		}
		s.mu.Unlock()
	}

	return ch, unsub
}

func (r *StreamRegistry) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.cleanup(time.Now().Add(-r.ttl))
	}
}

func (r *StreamRegistry) cleanup(cutoff time.Time) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, s := range r.sessions {
		s.mu.RLock()
		done := s.done
		s.mu.RUnlock()
		if done {
			delete(r.sessions, id)
		}
		_ = cutoff
	}
}
