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

	requestID := RequestIDFromContext(r.Context())
	incoming := len(req.Upsert)
	count, err := h.index.Sync(r.Context(), h.llm, req, h.cfg.MaxIndexChunks())
	if err != nil {
		logReq(requestID, "индекс sync workspace=%s ошибка: %v", req.WorkspaceID, err)
		writeBadRequest(w, err.Error())
		return
	}

	logReq(requestID, "индекс sync workspace=%s входящих=%d сохранено=%d", req.WorkspaceID, incoming, count)
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

	requestID := RequestIDFromContext(r.Context())
	resp, err := h.index.Search(r.Context(), h.llm, req)
	if err != nil {
		logReq(requestID, "поиск workspace=%s query=%q ошибка: %v", req.WorkspaceID, logPreview(req.Query, 60), err)
		h.mapRunnerError(w, err)
		return
	}

	if resp.Hits == nil {
		resp.Hits = []domain.SearchHit{}
	}

	logReq(requestID, "поиск workspace=%s query=%q mode=%s hits=%d", req.WorkspaceID, logPreview(req.Query, 60), req.Mode, len(resp.Hits))
	writeJSON(w, http.StatusOK, resp)
}
