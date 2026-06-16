package delivery

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/service"
)

func (h *Handler) enrichContextFromSearch(ctx context.Context, req *domain.ChatRequest) {
	service.EnrichContextFromSearch(ctx, h.index, h.llm, req)
}

func (h *Handler) chatSystemPrompt(req domain.ChatRequest, realMCPTools bool) string {
	system := ""
	if req.System != nil {
		system = strings.TrimSpace(*req.System)
	}

	if realMCPTools {
		return system
	}

	if h.mcp == nil || !h.mcp.Enabled() || req.Session == nil || req.Session.MCPEnabled == nil || !*req.Session.MCPEnabled {
		return system
	}

	block := h.mcp.ToolsPromptBlockForServers(req.Session.MCPServerIDs)
	if block == "" {
		return system
	}
	
	if system == "" {
		return block
	}

	return system + "\n\n" + block
}

func (h *Handler) agentPolicy() *service.AgentPolicy {
	if h.agent == nil {
		return nil
	}

	return h.agent.Policy()
}

func (h *Handler) handleQueue(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/v1/queue/jobs/") {
		h.handleQueueJob(w, r)
		return
	}

	if r.URL.Path == "/v1/queue" {
		h.handleQueueStats(w, r)
		return
	}

	writeJSON(w, http.StatusNotFound, domain.NewErrorResponse("not_found", fmt.Sprintf("path %q не найден", r.URL.Path)))
}
