package delivery

import (
	"net/http"
)

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	if h.metrics == nil {
		writeJSON(w, http.StatusOK, map[string]any{})
		return
	}

	snap := h.metrics.Snapshot(h.llm.RequestQueue(), h.ActiveStreams(), h.quota)
	writeJSON(w, http.StatusOK, snap)
}
