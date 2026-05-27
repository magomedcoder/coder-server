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
	"github.com/magomedcoder/coder-server/internal/contextbuilder"
	"github.com/magomedcoder/coder-server/internal/delivery"
	"github.com/magomedcoder/coder-server/internal/service"
)

type App struct {
	cfg     *config.Config
	llm     *service.LLMRunnerService
	handler *delivery.Handler
	streams *delivery.ActiveStreams
	metrics *service.Metrics
	server  *http.Server
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфиг не задан")
	}

	streams := delivery.NewActiveStreamsTracker()
	metrics := service.NewMetrics()
	llm, err := service.NewLLMRunnerService(cfg, metrics)
	if err != nil {
		return nil, fmt.Errorf("не удалось инициализировать клиент gen-runner: %w", err)
	}

	agent := service.NewAgentService(llm, cfg)
	index := service.NewRepoIndex()
	quota := service.NewTokenQuota(cfg.Quotas.MaxTokensPerDay)
	idempotency := service.NewIdempotencyStore(cfg.IdempotencyTTL())
	prefixCache := contextbuilder.NewPrefixCache(cfg.PromptCacheEntries())

	return &App{
		cfg:     cfg,
		llm:     llm,
		handler: delivery.NewHandler(cfg, llm, agent, index, quota, idempotency, prefixCache, streams, metrics),
		streams: streams,
		metrics: metrics,
	}, nil
}

func (a *App) Close() {
	if a == nil || a.llm == nil {
		return
	}

	if err := a.llm.Close(); err != nil {
		log.Printf("предупреждение: не удалось закрыть клиент runner: %v", err)
	}
}

func (a *App) Run() error {
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
		log.Printf("получен сигнал %v, graceful shutdown...", sig)
		return a.shutdown()
	}
}

func (a *App) shutdown() error {
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
		return fmt.Errorf("shutdown: %w", err)
	}

	log.Println("coder-server остановлен")
	return nil
}
