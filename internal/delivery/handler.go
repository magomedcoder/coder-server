package delivery

import (
	"context"
	"net/http"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/service"
	"github.com/magomedcoder/coder-server/pkg/contextbuilder"
)

type Handler struct {
	cfg           *config.Config
	llm           *service.LLMRunnerService
	agent         *service.AgentService
	chat          *service.ChatService
	chatSessions  *service.ChatSessionStore
	chatMCPLoop   *service.ChatMCPLoop
	index         *service.RepoIndex
	quota         *service.TokenQuota
	idempotency   *service.IdempotencyStore
	prefixCache   *contextbuilder.PrefixCache
	mcp           *service.MCPRegistry
	testSuggest   *service.TestSuggestService
	jobs          *service.JobStore
	sandbox       *service.CommandSandbox
	activeStreams *ActiveStreams
	metrics       *service.Metrics
}

func NewHandler(cfg *config.Config, llm *service.LLMRunnerService, agent *service.AgentService, chat *service.ChatService, chatSessions *service.ChatSessionStore, chatMCPLoop *service.ChatMCPLoop, index *service.RepoIndex, quota *service.TokenQuota, idempotency *service.IdempotencyStore, prefixCache *contextbuilder.PrefixCache, mcp *service.MCPRegistry, testSuggest *service.TestSuggestService, jobs *service.JobStore, sandbox *service.CommandSandbox, streams *ActiveStreams, metrics *service.Metrics) *Handler {
	return &Handler{
		cfg:           cfg,
		llm:           llm,
		agent:         agent,
		chat:          chat,
		chatSessions:  chatSessions,
		chatMCPLoop:   chatMCPLoop,
		index:         index,
		quota:         quota,
		idempotency:   idempotency,
		prefixCache:   prefixCache,
		mcp:           mcp,
		testSuggest:   testSuggest,
		jobs:          jobs,
		sandbox:       sandbox,
		activeStreams: streams,
		metrics:       metrics,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/queue", h.handleQueue)
	mux.HandleFunc("/v1/queue/", h.handleQueue)
	mux.HandleFunc("/v1/agent/run", h.handleAgentRun)
	mux.HandleFunc("/v1/health", h.handleHealth)
	mux.HandleFunc("/v1/health/live", h.handleHealthLive)
	mux.HandleFunc("/v1/health/ready", h.handleHealthReady)
	mux.HandleFunc("/v1/metrics", h.handleMetrics)
	mux.HandleFunc("/v1/models", h.handleModels)
	mux.HandleFunc("/v1/mcp/tools", h.handleMCPTools)
	mux.HandleFunc("/v1/mcp/call", h.handleMCPCall)
	mux.HandleFunc("/v1/index/sync", h.handleIndexSync)
	mux.HandleFunc("/v1/index/graph", h.handleIndexGraph)
	mux.HandleFunc("/v1/search", h.handleSearch)
	mux.HandleFunc("/v1/chat", h.handleChat)
	mux.HandleFunc("/v1/chat/stream", h.handleChatStream)
	mux.HandleFunc("/v1/chat/sessions/", h.handleChatSession)
	mux.HandleFunc("/v1/complete", h.handleComplete)
	mux.HandleFunc("/v1/edit", h.handleEdit)
	mux.HandleFunc("/v1/agent/step", h.handleAgentStep)
	mux.HandleFunc("/v1/agent/test-suggest", h.handleTestSuggest)
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

func (h *Handler) checkTokenQuota(estimate int64) bool {
	if h.quota == nil {
		return true
	}

	return h.quota.WouldAllow(estimate)
}

func (h *Handler) recordTokenUsage(prompt, completion int32) {
	if h.metrics != nil {
		h.metrics.RecordTokens(prompt, completion)
	}

	if h.quota != nil {
		total := int64(prompt) + int64(completion)
		h.quota.Record(total)
	}
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
