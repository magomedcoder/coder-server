package delivery

import (
	"context"
	"net/http"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/service"
)

type Handler struct {
	cfg           *config.Config
	llm           *service.LLMRunnerService
	agent         *service.AgentService
	activeStreams *ActiveStreams
	metrics       *service.Metrics
}

func NewHandler(cfg *config.Config, llm *service.LLMRunnerService, agent *service.AgentService, streams *ActiveStreams, metrics *service.Metrics) *Handler {
	return &Handler{cfg: cfg, llm: llm, agent: agent, activeStreams: streams, metrics: metrics}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/health", h.handleHealth)
	mux.HandleFunc("/v1/health/live", h.handleHealthLive)
	mux.HandleFunc("/v1/health/ready", h.handleHealthReady)
	mux.HandleFunc("/v1/metrics", h.handleMetrics)
	mux.HandleFunc("/v1/chat", h.handleChat)
	mux.HandleFunc("/v1/chat/stream", h.handleChatStream)
	mux.HandleFunc("/v1/agent/step", h.handleAgentStep)
}

func (h *Handler) ActiveStreams() int64 {
	if h == nil || h.activeStreams == nil {
		return 0
	}
	return h.activeStreams.Count()
}

func (h *Handler) ensureRunnerReady(ctx context.Context, w http.ResponseWriter) bool {
	return ensureRunnerReady(ctx, h.llm, w)
}

func (h *Handler) mapRunnerError(w http.ResponseWriter, err error) {
	mapRunnerError(w, err)
}

func (h *Handler) recordChatOK() {
	if h.metrics != nil {
		h.metrics.ChatRequests.Add(1)
	}
}

func (h *Handler) recordChatErr() {
	if h.metrics != nil {
		h.metrics.ChatRequests.Add(1)
		h.metrics.ChatErrors.Add(1)
	}
}

func (h *Handler) recordAgentOK() {
	if h.metrics != nil {
		h.metrics.AgentSteps.Add(1)
	}
}

func (h *Handler) recordAgentErr() {
	if h.metrics != nil {
		h.metrics.AgentSteps.Add(1)
		h.metrics.AgentErrors.Add(1)
	}
}
