package app

import (
	"fmt"
	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/delivery"
	"log"
	"net/http"

	"github.com/magomedcoder/coder-server/internal/service"
)

type App struct {
	cfg     *config.Config
	llm     *service.LLMRunnerService
	handler *delivery.Handler
}

func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфиг не задан")
	}

	llm, err := service.NewLLMRunnerService(cfg.RunnerAddr)
	if err != nil {
		return nil, fmt.Errorf("не удалось инициализировать клиент gen-runner: %w", err)
	}

	agent := service.NewAgentService(llm, cfg)

	return &App{
		cfg:     cfg,
		llm:     llm,
		handler: delivery.NewHandler(cfg, llm, agent),
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
	log.Println("Coder-server запущен")

	return http.ListenAndServe(addr, delivery.WithCORS(mux))
}
