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
