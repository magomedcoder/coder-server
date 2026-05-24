package service

import (
	"sync"

	"github.com/magomedcoder/coder-server/internal/domain"
)

type AgentSession struct {
	StepCount int
	Goal      string
}

type AgentSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*AgentSession
	maxSteps int
}

func NewAgentSessionStore(maxSteps int) *AgentSessionStore {
	if maxSteps <= 0 {
		maxSteps = 30
	}
	return &AgentSessionStore{
		sessions: make(map[string]*AgentSession),
		maxSteps: maxSteps,
	}
}

func (st *AgentSessionStore) BeginStep(sessionID, goal string) (domain.AgentStepResponse, bool) {
	if st == nil || sessionID == "" {
		return domain.AgentStepResponse{}, false
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	s, ok := st.sessions[sessionID]
	if !ok {
		s = &AgentSession{Goal: goal}
		st.sessions[sessionID] = s
	}
	if goal != "" {
		s.Goal = goal
	}

	s.StepCount++
	if s.StepCount > st.maxSteps {
		return domain.AgentStepResponse{
			Finish:  true,
			Summary: "Достигнут лимит шагов агента",
			Calls:   []domain.AgentToolCall{},
		}, true
	}

	return domain.AgentStepResponse{}, false
}

func (st *AgentSessionStore) Reset(sessionID string) {
	if st == nil || sessionID == "" {
		return
	}

	st.mu.Lock()
	delete(st.sessions, sessionID)
	st.mu.Unlock()
}
