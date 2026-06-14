package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func (h *Handler) handleComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req domain.CompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if !h.ensureRunnerReady(r.Context(), w) {
		return
	}

	resp, err := h.chat.InlineComplete(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "moderation layer") || strings.Contains(err.Error(), "секреты") {
			writeJSON(w, http.StatusBadRequest, domain.NewErrorResponse("invalid_request", err.Error()))
			return
		}

		h.mapRunnerError(w, err)
		return
	}

	h.recordChatOK()
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req domain.EditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if !h.ensureRunnerReady(r.Context(), w) {
		return
	}

	resp, err := h.chat.EditSelection(r.Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "moderation layer") || strings.Contains(err.Error(), "секреты") {
			writeJSON(w, http.StatusBadRequest, domain.NewErrorResponse("invalid_request", err.Error()))
			return
		}

		if strings.Contains(err.Error(), "обязателен") {
			writeBadRequest(w, err.Error())
			return
		}
		h.mapRunnerError(w, err)
		return
	}

	h.recordChatOK()
	writeJSON(w, http.StatusOK, resp)
}
