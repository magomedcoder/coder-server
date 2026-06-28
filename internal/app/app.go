package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/delivery"
	"github.com/magomedcoder/coder-server/internal/service"
	"github.com/magomedcoder/coder-server/pkg/contextbuilder"
)

type App struct {
	cfg       *config.Config
	llm       *service.LLMRunnerService
	handler   *delivery.Handler
	streams   *delivery.ActiveStreams
	metrics   *service.Metrics
	jobRunner *service.JobRunner
	jobCancel context.CancelFunc
	server    *http.Server
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфиг не задан")
	}

	streams := delivery.NewActiveStreamsTracker()
	metrics := service.NewMetrics()
	llm, err := service.NewLLMRunnerService(cfg, metrics)
	if err != nil {
		return nil, fmt.Errorf("не удалось инициализировать клиент lm-runner: %w", err)
	}

	mcp := service.NewMCPRegistry(cfg.MCP)
	qdrant := service.NewQdrantClient(cfg.Index.Qdrant.URL, cfg.Index.Qdrant.APIKey, cfg.Index.Qdrant.CollectionPrefix)
	index := service.NewRepoIndex(cfg.SearchWorkers(), qdrant)
	prefixCache := contextbuilder.NewPrefixCache(cfg.PromptCacheEntries())
	chat := service.NewChatService(cfg, llm, index, prefixCache)
	chatSessions := service.NewChatSessionStore(cfg.HistoryMaxMessages())
	chatMCPLoop := service.NewChatMCPLoop(llm, mcp, 0)
	agent := service.NewAgentService(llm, cfg, mcp)
	quota := service.NewTokenQuota(cfg.Quotas.MaxTokensPerDay)
	idempotency := service.NewIdempotencyStore(cfg.IdempotencyTTL())
	testSuggest := service.NewTestSuggestService(llm, cfg.ChatTimeoutSeconds())
	sandbox := service.NewCommandSandbox(cfg.Agent.Sandbox, cfg.Agent.AllowedCommands)

	var jobs *service.JobStore
	if cfg.PersistentQueueEnabled() {
		jobs, err = service.NewJobStore(cfg.Reliability.PersistentQueuePath, cfg.Reliability.PersistentQueueMax)
		if err != nil {
			return nil, fmt.Errorf("постоянная очередь: %w", err)
		}
	}

	jobRunner := service.NewJobRunner(jobs, llm.RequestQueue(), chat, agent)

	app := &App{
		cfg:       cfg,
		llm:       llm,
		handler:   delivery.NewHandler(cfg, llm, agent, chat, chatSessions, chatMCPLoop, index, quota, idempotency, prefixCache, mcp, testSuggest, jobs, sandbox, streams, metrics),
		streams:   streams,
		metrics:   metrics,
		jobRunner: jobRunner,
	}
	app.logStartup(mcp, jobs)
	return app, nil
}

func (a *App) logStartup(mcp *service.MCPRegistry, jobs *service.JobStore) {
	if a == nil || a.cfg == nil {
		return
	}

	cfg := a.cfg
	runners := 0
	for _, r := range cfg.Runners {
		if r.Enabled == nil || *r.Enabled {
			runners++
		}
	}

	mcpCount := 0
	if mcp != nil {
		mcpCount = mcp.ServerCount()
	}

	log.Printf("инициализация: runners=%d mcp_серверов=%d budget_токенов=%d история=%d moderation=%v auth=%v rate_limit=%v", runners, mcpCount, cfg.ContextTokenBudget(), cfg.HistoryMaxMessages(), cfg.ModerationEnabled(), cfg.AuthEnabled(), cfg.RateLimitEnabled())

	if cfg.Quotas.MaxTokensPerDay > 0 {
		log.Printf("инициализация: квота токенов/день=%d", cfg.Quotas.MaxTokensPerDay)
	}

	if cfg.PersistentQueueEnabled() {
		log.Printf("инициализация: постоянная очередь path=%s max=%d", cfg.Reliability.PersistentQueuePath, cfg.Reliability.PersistentQueueMax)
	}

	if jobs != nil {
		log.Printf("инициализация: фоновый обработчик очереди запущен")
	}

	if cfg.QdrantEnabled() {
		log.Printf("инициализация: qdrant url=%s", cfg.Index.Qdrant.URL)
	}
}

func (a *App) Close() {
	if a == nil {
		return
	}

	if a.jobCancel != nil {
		a.jobCancel()
	}

	if a.llm == nil {
		return
	}

	if err := a.llm.Close(); err != nil {
		log.Printf("предупреждение: не удалось закрыть клиент runner: %v", err)
	}
}

func (a *App) Run() error {
	if a.jobRunner != nil {
		ctx, cancel := context.WithCancel(context.Background())
		a.jobCancel = cancel
		go a.jobRunner.RunLoop(ctx)
		log.Println("фоновый цикл очереди задач запущен")
	}

	mux := http.NewServeMux()
	a.handler.Register(mux)

	addr := a.cfg.ListenAddr()
	inner := delivery.WithMiddleware(a.cfg, a.streams, a.metrics, mux)
	if a.cfg.RateLimitEnabled() {
		inner = delivery.WithRateLimit(a.cfg.RateLimit.RequestsPerMinute, inner)
	}
	handler := delivery.WithCORS(inner)

	a.server = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("coder-server слушает %s", addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-sigCh:
		log.Printf("получен сигнал %v, корректное завершение...", sig)
		return a.shutdown()
	}
}

func (a *App) shutdown() error {
	if a.jobCancel != nil {
		a.jobCancel()
	}

	if a.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deadline := time.Now().Add(30 * time.Second)
	for a.handler.ActiveStreams() > 0 && time.Now().Before(deadline) {
		log.Printf("ожидание завершения %d активных SSE-потоков...", a.handler.ActiveStreams())
		time.Sleep(500 * time.Millisecond)
	}

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("завершение работы: %w", err)
	}

	log.Println("coder-server остановлен")
	return nil
}
