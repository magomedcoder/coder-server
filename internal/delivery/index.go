package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func (h *Handler) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	models, err := h.llm.GetModels(r.Context())
	if err != nil {
		h.mapRunnerError(w, err)
		return
	}

	if models == nil {
		models = []string{}
	}

	writeJSON(w, http.StatusOK, domain.ModelsResponse{Models: models})
}

func (h *Handler) handleIndexSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req domain.IndexSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if h.index == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("internal_error", "индекс не инициализирован"))
		return
	}

	if strings.TrimSpace(req.WorkspaceID) == "" {
		writeBadRequest(w, "workspace_id обязателен")
		return
	}

	count, err := h.index.Sync(req, h.cfg.MaxIndexChunks())
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, domain.IndexSyncResponse{Chunks: count})
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req domain.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if strings.TrimSpace(req.WorkspaceID) == "" || strings.TrimSpace(req.Query) == "" {
		writeBadRequest(w, "workspace_id и query обязательны")
		return
	}

	if h.index == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("internal_error", "индекс не инициализирован"))
		return
	}

	resp, err := h.index.Search(r.Context(), h.llm, req)
	if err != nil {
		h.mapRunnerError(w, err)
		return
	}

	if resp.Hits == nil {
		resp.Hits = []domain.SearchHit{}
	}

	writeJSON(w, http.StatusOK, resp)
}
