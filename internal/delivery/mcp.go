package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func (h *Handler) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}

	if h.mcp == nil || !h.mcp.Enabled() {
		writeJSON(w, http.StatusOK, domain.MCPToolsResponse{Tools: []domain.MCPToolInfo{}})
		return
	}

	_ = h.mcp.Refresh(r.Context())
	tools := h.mcp.ListTools()
	if tools == nil {
		tools = []domain.MCPToolInfo{}
	}

	writeJSON(w, http.StatusOK, domain.MCPToolsResponse{Tools: tools})
}

func (h *Handler) handleMCPCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	if h.mcp == nil || !h.mcp.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("service_unavailable", "MCP не настроен"))
		return
	}

	var req domain.MCPCallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	result, err := h.mcp.Call(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, domain.NewErrorResponse("mcp_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, domain.MCPCallResponse{Result: result})
}

func (h *Handler) handleIndexGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "GET, POST")
		return
	}

	var req domain.IndexGraphRequest
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "некорректное тело JSON")
			return
		}
	} else {
		q := r.URL.Query()
		req.WorkspaceID = q.Get("workspace_id")
		req.Path = q.Get("path")
		req.Symbol = q.Get("symbol")
	}

	if req.WorkspaceID == "" {
		writeBadRequest(w, "workspace_id обязателен")
		return
	}
	if h.index == nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("internal_error", "индекс не инициализирован"))
		return
	}

	writeJSON(w, http.StatusOK, h.index.Graph(req.WorkspaceID, req.Path, req.Symbol))
}

func (h *Handler) handleTestSuggest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var req domain.TestSuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "некорректное тело JSON")
		return
	}

	if !h.ensureRunnerReady(r.Context(), w) {
		return
	}

	resp, err := h.testSuggest.Suggest(r.Context(), req)
	if err != nil {
		h.mapRunnerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
