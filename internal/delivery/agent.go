package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func (h *Handler) handleAgentStep(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req domain.AgentStepRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if !h.ensureRunnerReady(r.Context(), w) {
		return
	}

	resp, err := h.agent.Step(r.Context(), req)
	if err != nil {
		mapRunnerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
