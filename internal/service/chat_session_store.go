package service

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/magomedcoder/coder-server/internal/domain"
)

type ChatSessionRecord struct {
	Messages  []domain.ChatMessage
	UpdatedAt time.Time
}

type ChatSessionStore struct {
	mu          sync.Mutex
	sessions    map[string]*ChatSessionRecord
	maxMessages int
}

func NewChatSessionStore(maxMessages int) *ChatSessionStore {
	if maxMessages <= 0 {
		maxMessages = 200
	}
	return &ChatSessionStore{
		sessions:    make(map[string]*ChatSessionRecord),
		maxMessages: maxMessages,
	}
}

func (st *ChatSessionStore) ResolveSessionID(req *domain.ChatRequest) string {
	if req == nil {
		return ""
	}

	if req.Session != nil {
		if id := strings.TrimSpace(req.Session.SessionID); id != "" {
			return id
		}
	}

	return "chat-" + uuid.NewString()
}

func (st *ChatSessionStore) Merge(sessionID string, incoming []domain.ChatMessage) []domain.ChatMessage {
	if st == nil || sessionID == "" {
		return incoming
	}

	st.mu.Lock()
	rec, ok := st.sessions[sessionID]
	st.mu.Unlock()
	if !ok || rec == nil || len(rec.Messages) == 0 {
		return incoming
	}

	stored := append([]domain.ChatMessage(nil), rec.Messages...)
	if len(incoming) >= len(stored) {
		return incoming
	}

	if len(incoming) == 0 {
		return stored
	}

	last := incoming[len(incoming)-1]
	if last.Role != "user" {
		return stored
	}

	if len(stored) > 0 && stored[len(stored)-1].Role == "user" && stored[len(stored)-1].Content == last.Content {
		return stored
	}

	out := append(stored, last)
	return trimChatSessionMessages(out, st.maxMessages)
}

func (st *ChatSessionStore) Record(sessionID string, messages []domain.ChatMessage) {
	if st == nil || sessionID == "" || len(messages) == 0 {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	st.sessions[sessionID] = &ChatSessionRecord{
		Messages:  trimChatSessionMessages(append([]domain.ChatMessage(nil), messages...), st.maxMessages),
		UpdatedAt: time.Now(),
	}
}

func (st *ChatSessionStore) Get(sessionID string) ([]domain.ChatMessage, bool) {
	if st == nil || sessionID == "" {
		return nil, false
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	rec, ok := st.sessions[sessionID]
	if !ok || rec == nil || len(rec.Messages) == 0 {
		return nil, false
	}

	out := append([]domain.ChatMessage(nil), rec.Messages...)
	return out, true
}

func trimChatSessionMessages(messages []domain.ChatMessage, max int) []domain.ChatMessage {
	if max <= 0 || len(messages) <= max {
		return messages
	}

	return messages[len(messages)-max:]
}
