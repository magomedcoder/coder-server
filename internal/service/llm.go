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
	pool    *llmrunner.Pool
	reg     *llmrunner.Registry
	breaker *CircuitBreaker
	retries int
	streams *StreamRegistry
	queue   *RequestQueue
	metrics *Metrics
}

func NewLLMRunnerService(cfg *config.Config, metrics *Metrics) (*LLMRunnerService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("конфиг не задан")
	}

	reg := llmrunner.NewRegistry(cfg.RunnerStates())
	pool := llmrunner.NewPool(reg)

	return &LLMRunnerService{
		pool:    pool,
		reg:     reg,
		breaker: NewCircuitBreaker(cfg.Reliability.CircuitBreakerFailures, cfg.CircuitBreakerCooldown()),
		retries: cfg.Reliability.RunnerRetries,
		streams: NewStreamRegistry(cfg.SSEBufferTTL()),
		queue:   NewRequestQueue(cfg.Reliability.MaxConcurrentRequests, cfg.QueueWaitTimeout()),
		metrics: metrics,
	}, nil
}

func (s *LLMRunnerService) StreamRegistry() *StreamRegistry {
	if s == nil {
		return nil
	}
	return s.streams
}

func (s *LLMRunnerService) RequestQueue() *RequestQueue {
	if s == nil {
		return nil
	}
	return s.queue
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
		if s.breaker != nil && !s.breaker.Allow(addr) {
			continue
		}
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
		if s.breaker != nil && !s.breaker.Allow(addr) {
			continue
		}
		probe := s.pool.ProbeLLMRunner(ctx, addr)
		if probe.Connected && probe.LoadedModel != nil && probe.LoadedModel.Loaded {
			return probe, addr, nil
		}
	}

	for _, addr := range addrs {
		probe := s.pool.ProbeLLMRunner(ctx, addr)
		return probe, addr, nil
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

	if s.queue != nil {
		if err := s.queue.Acquire(ctx); err != nil {
			return nil, err
		}
	}

	ch, err := s.sendMessageWithRetry(ctx, messages, stopSequences, timeoutSeconds, genParams)
	if err != nil {
		if s.queue != nil {
			s.queue.Release()
		}
		return nil, err
	}

	return forwardWithQueueRelease(ch, s.queue), nil
}

func forwardWithQueueRelease(in <-chan gendomain.LLMStreamChunk, q *RequestQueue) chan gendomain.LLMStreamChunk {
	out := make(chan gendomain.LLMStreamChunk, 100)
	go func() {
		defer close(out)
		if q != nil {
			defer q.Release()
		}
		for chunk := range in {
			out <- chunk
		}
	}()
	return out
}

func (s *LLMRunnerService) sendMessageWithRetry(
	ctx context.Context,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
) (chan gendomain.LLMStreamChunk, error) {
	addrs := s.eligibleRunners()
	if len(addrs) == 0 {
		ch, err := s.pool.SendMessage(ctx, messages, stopSequences, timeoutSeconds, genParams)
		if err != nil {
			return nil, mapPoolError(err)
		}
		return ch, nil
	}

	var lastErr error
	attempts := len(addrs)
	if s.retries > 0 && s.retries < attempts {
		attempts = s.retries
	}

	for i := 0; i < attempts; i++ {
		addr := addrs[i%len(addrs)]
		ch, err := s.pool.SendMessageOnRunner(ctx, addr, messages, stopSequences, timeoutSeconds, genParams)
		if err == nil {
			if s.breaker != nil {
				s.breaker.RecordSuccess(addr)
			}
			return ch, nil
		}

		lastErr = err
		if s.breaker != nil {
			s.breaker.RecordFailure(addr)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	if lastErr != nil {
		return nil, mapPoolError(lastErr)
	}
	return nil, fmt.Errorf("gen-runner недоступен")
}

func (s *LLMRunnerService) eligibleRunners() []string {
	if s == nil || s.reg == nil {
		return nil
	}

	all := s.reg.GetEnabledAddresses()
	out := make([]string, 0, len(all))
	for _, addr := range all {
		if s.breaker != nil && !s.breaker.Allow(addr) {
			continue
		}
		out = append(out, addr)
	}
	return out
}

type CollectResult struct {
	Content string
	Usage   *gendomain.StreamTokenUsage
}

func (s *LLMRunnerService) CollectMessage(
	ctx context.Context,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
) (CollectResult, error) {
	ch, err := s.SendMessage(ctx, messages, stopSequences, timeoutSeconds, genParams)
	if err != nil {
		return CollectResult{}, err
	}

	var full strings.Builder
	var usage *gendomain.StreamTokenUsage
	for chunk := range ch {
		if chunk.Content != "" {
			full.WriteString(chunk.Content)
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	if s.metrics != nil && usage != nil {
		s.metrics.RecordTokens(usage.PromptTokens, usage.CompletionTokens)
	}

	return CollectResult{Content: full.String(), Usage: usage}, nil
}

func (s *LLMRunnerService) GetModels(ctx context.Context) ([]string, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("pool не инициализирован")
	}

	return s.pool.GetModels(ctx)
}

func (s *LLMRunnerService) Embed(ctx context.Context, text string) ([]float32, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("pool не инициализирован")
	}

	return s.pool.Embed(ctx, text)
}

func mapPoolError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gendomain.ErrRunnerModelNotLoaded) {
		return err
	}
	if errors.Is(err, ErrQueueTimeout) {
		return err
	}
	return err
}
