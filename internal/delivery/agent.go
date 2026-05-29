package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/security"
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

	requestID := resolveRequestID(r.Context(), optionalString(req.RequestID))
	if requestID != "" && h.idempotency != nil {
		if status, body, ok := h.idempotency.Get("agent:" + requestID); ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Idempotent-Replay", "true")
			w.WriteHeader(status)
			_, _ = w.Write(body)
			return
		}
	}

	if h.cfg.ModerationEnabled() {
		texts := []string{req.Goal}
		for _, obs := range req.Observations {
			if obs.Error != "" {
				texts = append(texts, obs.Error)
			}
			for _, v := range obs.Result {
				if s, ok := v.(string); ok {
					texts = append(texts, s)
				}
			}
		}
		if security.ScanMessages(texts) {
			h.recordAgentErr()
			writeJSON(w, http.StatusBadRequest, domain.NewErrorResponse("prompt_injection", "запрос отклонён moderation layer"))
			return
		}
	}

	if !h.ensureRunnerReady(r.Context(), w) {
		return
	}

	resp, err := h.agent.Step(r.Context(), req)
	if err != nil {
		h.recordAgentErr()
		h.mapRunnerErrorWithQueue(w, err, "agent_step", requestID, req)
		return
	}

	h.recordAgentOK()
	if requestID != "" && h.idempotency != nil {
		if body, err := json.Marshal(resp); err == nil {
			h.idempotency.Put("agent:"+requestID, http.StatusOK, body)
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func optionalString(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
