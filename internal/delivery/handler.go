package delivery

import (
	"context"
	"github.com/magomedcoder/tce-server/internal/config"
	"net/http"
	"strings"

	"github.com/magomedcoder/tce-server/internal/domain"
	"github.com/magomedcoder/tce-server/internal/service"
)

type Handler struct {
	cfg   *config.Config
	llm   *service.LLMRunnerService
	agent *service.AgentService
}

func NewHandler(cfg *config.Config, llm *service.LLMRunnerService, agent *service.AgentService) *Handler {
	return &Handler{cfg: cfg, llm: llm, agent: agent}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/health", h.handleHealth)
	mux.HandleFunc("/v1/chat", h.handleChat)
	mux.HandleFunc("/v1/agent/step", h.handleAgentStep)
}

func (h *Handler) ensureRunnerReady(ctx context.Context, w http.ResponseWriter) bool {
	ok, err := h.llm.CheckConnection(ctx)
	if err != nil || !ok {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("internal_error", "gen-runner недоступен"))
		return false
	}

	if err := h.llm.ModelReady(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("internal_error", err.Error()))
		return false
	}

	return true
}

func mapRunnerError(w http.ResponseWriter, err error) {
	msg := err.Error()
	if strings.Contains(msg, "модель не загружена") {
		writeJSON(w, http.StatusServiceUnavailable, domain.NewErrorResponse("internal_error", msg))
		return
	}
	writeJSON(w, http.StatusInternalServerError, domain.NewErrorResponse("internal_error", "ошибка генерации"))
}
