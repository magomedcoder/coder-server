package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/magomedcoder/tce-server/internal/service"
)

type App struct {
	llm        *service.LLMRunnerService
	runnerAddr string
	host       string
}

func NewFromEnv() (*App, error) {
	runnerAddr := "127.0.0.1:50052"
	host := "127.0.0.1:8000"

	llmSvc, err := service.NewLLMRunnerService(runnerAddr)
	if err != nil {
		return nil, fmt.Errorf("не удалось инициализировать клиент gen-runner: %w", err)
	}

	return &App{
		llm:        llmSvc,
		runnerAddr: runnerAddr,
		host:       host,
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
	mux.HandleFunc("/v1/health", a.handleHealth)
	mux.HandleFunc("/v1/chat", a.handleChat)
	mux.HandleFunc("/v1/agent/step", a.handleAgentStep)

	log.Printf("Tce-server запущен на %s", a.host)
	log.Printf("gen-runner: %s", a.runnerAddr)
	log.Printf("проверка здоровья: GET  http://%s/v1/health", a.host)
	log.Printf("чат:               POST http://%s/v1/chat", a.host)
	log.Printf("шаг агента:        POST http://%s/v1/agent/step", a.host)

	return http.ListenAndServe(a.host, withCORS(mux))
}

func (a *App) ensureRunnerReady(ctx context.Context, w http.ResponseWriter) bool {
	ok, err := a.llm.CheckConnection(ctx)
	if err != nil || !ok {
		writeJSON(w, http.StatusServiceUnavailable, errorBody("internal_error", "gen-runner недоступен"))
		return false
	}

	if err := a.llm.ModelReady(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, errorBody("internal_error", err.Error()))
		return false
	}

	return true
}

func mapRunnerError(w http.ResponseWriter, err error) {
	msg := err.Error()
	if strings.Contains(msg, "модель не загружена") {
		writeJSON(w, http.StatusServiceUnavailable, errorBody("internal_error", msg))
		return
	}

	writeJSON(w, http.StatusInternalServerError, errorBody("internal_error", "ошибка генерации"))
}
