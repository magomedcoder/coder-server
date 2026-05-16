package delivery

import (
	"net/http"

	"github.com/magomedcoder/tce-server/internal/domain"
)

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	ok, err := h.llm.CheckConnection(r.Context())
	if err != nil || !ok {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("internal_error", "gen-runner недоступен"))
		return
	}

	loaded := false
	if ok {
		if err := h.llm.ModelReady(r.Context()); err == nil {
			loaded = true
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok": ok && loaded,
	})
}
