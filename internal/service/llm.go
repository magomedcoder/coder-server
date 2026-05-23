package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/config"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/llmrunner"
)

type LLMRunnerService struct {
	pool *llmrunner.Pool
	reg  *llmrunner.Registry
}

func NewLLMRunnerService(cfg *config.Config) (*LLMRunnerService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфиг не задан")
	}

	reg := llmrunner.NewRegistry(cfg.RunnerStates())
	pool := llmrunner.NewPool(reg)

	return &LLMRunnerService{pool: pool, reg: reg}, nil
}

func (s *LLMRunnerService) Close() error {
	if s == nil || s.pool == nil {
		return nil
	}

	return s.pool.Close()
}

func (s *LLMRunnerService) Registry() *llmrunner.Registry {
	if s == nil {
		return nil
	}
	return s.reg
}

func (s *LLMRunnerService) CheckConnection(ctx context.Context) (bool, error) {
	if s == nil || s.pool == nil {
		return false, fmt.Errorf("pool не инициализирован")
	}
	return s.pool.CheckConnection(ctx)
}

func (s *LLMRunnerService) ModelReady(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("pool не инициализирован")
	}

	addrs := s.reg.GetEnabledAddresses()
	if len(addrs) == 0 {
		return fmt.Errorf("нет включённых gen-runner")
	}

	for _, addr := range addrs {
		if err := s.pool.RequireChatModelLoaded(ctx, addr); err == nil {
			return nil
		}
	}

	return gendomain.ErrRunnerModelNotLoaded
}

func (s *LLMRunnerService) ProbeBestRunner(ctx context.Context) (llmrunner.RunnerProbeResult, string, error) {
	if s == nil || s.pool == nil {
		return llmrunner.RunnerProbeResult{}, "", fmt.Errorf("pool не инициализирован")
	}

	addrs := s.reg.GetEnabledAddresses()
	for _, addr := range addrs {
		probe := s.pool.ProbeLLMRunner(ctx, addr)
		if probe.Connected && probe.LoadedModel != nil && probe.LoadedModel.Loaded {
			return probe, addr, nil
		}
	}

	if len(addrs) > 0 {
		probe := s.pool.ProbeLLMRunner(ctx, addrs[0])
		return probe, addrs[0], nil
	}

	return llmrunner.RunnerProbeResult{}, "", fmt.Errorf("нет доступных gen-runner")
}

func (s *LLMRunnerService) ChatHints() llmrunner.RunnerCoreHints {
	if s == nil || s.reg == nil {
		return llmrunner.DefaultRunnerCoreHints()
	}
	return s.reg.AggregateChatHints()
}

func (s *LLMRunnerService) SendMessage(
	ctx context.Context,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
) (chan gendomain.LLMStreamChunk, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("pool не инициализирован")
	}

	ch, err := s.pool.SendMessage(ctx, messages, stopSequences, timeoutSeconds, genParams)
	if err != nil {
		return nil, mapPoolError(err)
	}
	return ch, nil
}

func (s *LLMRunnerService) CollectMessage(
	ctx context.Context,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
) (string, error) {
	ch, err := s.SendMessage(ctx, messages, stopSequences, timeoutSeconds, genParams)
	if err != nil {
		return "", err
	}

	var full strings.Builder
	for chunk := range ch {
		if chunk.Content != "" {
			full.WriteString(chunk.Content)
		}
	}

	return full.String(), nil
}

func mapPoolError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gendomain.ErrRunnerModelNotLoaded) {
		return err
	}
	return err
}
