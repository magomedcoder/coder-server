package delivery

import (
	"context"
	"net/http"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/service"
)

type Handler struct {
	cfg     *config.Config
	llm     *service.LLMRunnerService
	agent   *service.AgentService
	streams *ActiveStreams
}

func NewHandler(cfg *config.Config, llm *service.LLMRunnerService, agent *service.AgentService, streams *ActiveStreams) *Handler {
	return &Handler{cfg: cfg, llm: llm, agent: agent, streams: streams}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/health", h.handleHealth)
	mux.HandleFunc("/v1/chat", h.handleChat)
	mux.HandleFunc("/v1/agent/step", h.handleAgentStep)
}

func (h *Handler) ActiveStreams() int64 {
	if h == nil || h.streams == nil {
		return 0
	}
	return h.streams.Count()
}

func (h *Handler) ensureRunnerReady(ctx context.Context, w http.ResponseWriter) bool {
	return ensureRunnerReady(ctx, h.llm, w)
}

func (h *Handler) mapRunnerError(w http.ResponseWriter, err error) {
	mapRunnerError(w, err)
}
