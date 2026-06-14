package service

import (
	"fmt"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/pkg/llmclient"
)

type LLMRunnerService = llmclient.Service
type CollectResult = llmclient.CollectResult

func NewLLMRunnerService(cfg *config.Config, metrics *Metrics) (*LLMRunnerService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфиг не задан")
	}

	opts := cfg.LLMClientOptions()
	if metrics != nil {
		opts.OnTokens = metrics.RecordTokens
	}

	return llmclient.New(opts)
}
