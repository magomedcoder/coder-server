package ssestream

import (
	"sync"
	"time"
)

type Event struct {
	ID    int
	Event string
	Data  string
}

type Session struct {
	requestID string
	events    []Event
	done      bool
	subs      map[int]chan Event
	nextSubID int
	mu        sync.RWMutex
}

type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	ttl      time.Duration
}

func NewRegistry(ttl time.Duration) *Registry {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	r := &Registry{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}

	go r.cleanupLoop()

	return r
}

func (r *Registry) Start(requestID string) *Session {
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

	s := &Session{
		requestID: requestID,
		subs:      make(map[int]chan Event),
	}

	r.sessions[requestID] = s

	return s
}

func (r *Registry) Get(requestID string) (*Session, bool) {
	if r == nil || requestID == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	s, ok := r.sessions[requestID]
	return s, ok
}

func (s *Session) Append(event Event) {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.events = append(s.events, event)
	subs := make([]chan Event, 0, len(s.subs))
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

func (s *Session) MarkDone() {
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

func (s *Session) IsDone() bool {
	if s == nil {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.done
}

func (s *Session) EventsAfter(lastEventID int) []Event {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Event, 0)
	for _, e := range s.events {
		if e.ID > lastEventID {
			out = append(out, e)
		}
	}

	return out
}

func (s *Session) Subscribe() (<-chan Event, func()) {
	if s == nil {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}

	s.mu.Lock()
	id := s.nextSubID
	s.nextSubID++
	ch := make(chan Event, 64)
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

func (r *Registry) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.cleanup(time.Now().Add(-r.ttl))
	}
}

func (r *Registry) cleanup(cutoff time.Time) {
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
