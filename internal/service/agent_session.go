package service

import (
	"fmt"
	"strings"
	"sync"

	"github.com/magomedcoder/coder-server/internal/domain"
)

type AgentSession struct {
	StepCount   int
	Goal        string
	LastSummary string
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

func (st *AgentSessionStore) BeginStep(sessionID, goal string) (step int, limited domain.AgentStepResponse, stop bool) {
	if st == nil || sessionID == "" {
		return 0, domain.AgentStepResponse{}, false
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
	step = s.StepCount

	if s.StepCount > st.maxSteps {
		return step, domain.AgentStepResponse{
			Finish:  true,
			Summary: "Достигнут лимит шагов агента",
			Calls:   []domain.AgentToolCall{},
			Step:    step,
		}, true
	}

	return step, domain.AgentStepResponse{}, false
}

func (st *AgentSessionStore) Reset(sessionID string) {
	if st == nil || sessionID == "" {
		return
	}

	st.mu.Lock()
	delete(st.sessions, sessionID)
	st.mu.Unlock()
}

func (st *AgentSessionStore) SetSummary(sessionID, summary string) {
	if st == nil || sessionID == "" {
		return
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	s, ok := st.sessions[sessionID]
	if !ok {
		return
	}

	s.LastSummary = summary
}

func (st *AgentSessionStore) ContextHint(sessionID string) string {
	if st == nil || sessionID == "" {
		return ""
	}

	st.mu.Lock()
	defer st.mu.Unlock()
	s, ok := st.sessions[sessionID]
	if !ok {
		return ""
	}

	if s.LastSummary != "" {
		return fmt.Sprintf("step %d/%d, last summary: %s", s.StepCount, st.maxSteps, s.LastSummary)
	}

	if s.Goal != "" {
		return fmt.Sprintf("step %d/%d, goal: %s", s.StepCount, st.maxSteps, s.Goal)
	}

	return fmt.Sprintf("step %d/%d", s.StepCount, st.maxSteps)
}
