package delivery

import (
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func (h *Handler) handleChatSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	if h.chatSessions == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", "хранилище сессий не инициализировано"))
		return
	}

	sessionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/v1/chat/sessions/"))
	if sessionID == "" || strings.Contains(sessionID, "/") {
		writeBadRequest(w, "session_id обязателен")
		return
	}

	messages, ok := h.chatSessions.Get(sessionID)
	if !ok {
		writeJSON(w, http.StatusNotFound, domain.NewErrorResponse("not_found", "сессия не найдена"))
		return
	}

	writeJSON(w, http.StatusOK, domain.ChatSessionResponse{
		SessionID: sessionID,
		Messages:  messages,
	})
}
