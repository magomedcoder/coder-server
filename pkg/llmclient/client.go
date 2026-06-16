package llmclient

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/pkg/circuitbreaker"
	"github.com/magomedcoder/coder-server/pkg/requestqueue"
	"github.com/magomedcoder/coder-server/pkg/ssestream"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/llmrunner"
	"github.com/magomedcoder/gen/pkg/runnerresolve"
)

type Service struct {
	llm      gendomain.LLMRepository
	pool     *llmrunner.Pool
	reg      *llmrunner.Registry
	breaker  *circuitbreaker.Breaker
	retries  int
	streams  *ssestream.Registry
	queue    *requestqueue.Queue
	onTokens func(prompt, completion int32)
}

func New(opts Options) (*Service, error) {
	if len(opts.RunnerStates) == 0 {
		return nil, fmt.Errorf("llmclient: нужен хотя бы один runner")
	}

	reg := llmrunner.NewRegistry(opts.RunnerStates)
	pool := llmrunner.NewPool(reg)
	rel := opts.Reliability

	return &Service{
		llm:      pool,
		pool:     pool,
		reg:      reg,
		breaker:  circuitbreaker.New(rel.CircuitBreakerFailures, rel.CircuitBreakerCooldown),
		retries:  rel.RunnerRetries,
		streams:  ssestream.NewRegistry(opts.SSEBufferTTL),
		queue:    requestqueue.New(rel.MaxConcurrentRequests, rel.QueueWaitTimeout),
		onTokens: opts.OnTokens,
	}, nil
}

func (s *Service) StreamRegistry() *ssestream.Registry {
	if s == nil {
		return nil
	}

	return s.streams
}

func (s *Service) RequestQueue() *requestqueue.Queue {
	if s == nil {
		return nil
	}

	return s.queue
}

func (s *Service) CircuitSnapshot() map[string]string {
	if s == nil || s.breaker == nil {
		return nil
	}

	return s.breaker.Snapshot()
}

func (s *Service) Close() error {
	if s == nil || s.pool == nil {
		return nil
	}

	return s.pool.Close()
}

func (s *Service) Registry() *llmrunner.Registry {
	if s == nil {
		return nil
	}

	return s.reg
}

func (s *Service) CheckConnection(ctx context.Context) (bool, error) {
	if s == nil || s.llm == nil {
		return false, fmt.Errorf("pool не инициализирован")
	}

	return s.llm.CheckConnection(ctx)
}

func (s *Service) ModelReady(ctx context.Context) error {
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

func (s *Service) ProbeBestRunner(ctx context.Context) (llmrunner.RunnerProbeResult, string, error) {
	if s == nil || s.pool == nil || s.reg == nil {
		return llmrunner.RunnerProbeResult{}, "", fmt.Errorf("pool не инициализирован")
	}

	entries := runnerresolve.ListEnabledEntries(ctx, s.reg, s.pool)
	for _, entry := range entries {
		if s.breaker != nil && !s.breaker.Allow(entry.Address) {
			continue
		}

		if !entry.Connected {
			continue
		}

		probe := s.pool.ProbeLLMRunner(ctx, entry.Address)
		if probe.LoadedModel != nil && probe.LoadedModel.Loaded {
			return probe, entry.Address, nil
		}
	}

	for _, entry := range entries {
		if s.breaker != nil && !s.breaker.Allow(entry.Address) {
			continue
		}

		probe := s.pool.ProbeLLMRunner(ctx, entry.Address)
		return probe, entry.Address, nil
	}

	return llmrunner.RunnerProbeResult{}, "", fmt.Errorf("нет доступных gen-runner")
}

func (s *Service) ChatHints() llmrunner.RunnerCoreHints {
	if s == nil || s.reg == nil {
		return llmrunner.DefaultRunnerCoreHints()
	}

	return s.reg.AggregateChatHints()
}

func (s *Service) SendMessage(
	ctx context.Context,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
) (chan gendomain.LLMStreamChunk, error) {
	if s == nil || s.llm == nil {
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

func (s *Service) SendMessageOnRunner(
	ctx context.Context,
	runnerAddr string,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
) (chan gendomain.LLMStreamChunk, error) {
	if s == nil || s.llm == nil {
		return nil, fmt.Errorf("pool не инициализирован")
	}

	addr := strings.TrimSpace(runnerAddr)
	if addr == "" {
		return nil, fmt.Errorf("runner address пуст")
	}

	if s.queue != nil {
		if err := s.queue.Acquire(ctx); err != nil {
			return nil, err
		}
	}

	ch, err := s.llm.SendMessageOnRunner(ctx, addr, messages, stopSequences, timeoutSeconds, genParams)
	if err != nil {
		if s.queue != nil {
			s.queue.Release()
		}
		return nil, mapPoolError(err)
	}

	return forwardWithQueueRelease(ch, s.queue), nil
}

func forwardWithQueueRelease(in <-chan gendomain.LLMStreamChunk, q *requestqueue.Queue) chan gendomain.LLMStreamChunk {
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

func (s *Service) sendMessageWithRetry(
	ctx context.Context,
	messages []*gendomain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *gendomain.GenerationParams,
) (chan gendomain.LLMStreamChunk, error) {
	addrs := s.eligibleRunners()
	if len(addrs) == 0 {
		ch, err := s.llm.SendMessage(ctx, messages, stopSequences, timeoutSeconds, genParams)
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
		ch, err := s.llm.SendMessageOnRunner(ctx, addr, messages, stopSequences, timeoutSeconds, genParams)
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

func (s *Service) eligibleRunners() []string {
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
	Content   string
	Reasoning string
	Usage     *gendomain.StreamTokenUsage
}

func (s *Service) CollectMessage(
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

	var content strings.Builder
	var reasoning strings.Builder
	var usage *gendomain.StreamTokenUsage
	for chunk := range ch {
		if chunk.ReasoningContent != "" {
			reasoning.WriteString(chunk.ReasoningContent)
		}

		if chunk.Content != "" {
			content.WriteString(chunk.Content)
		}

		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}

	if s.onTokens != nil && usage != nil {
		s.onTokens(usage.PromptTokens, usage.CompletionTokens)
	}

	return CollectResult{
		Content:   content.String(),
		Reasoning: reasoning.String(),
		Usage:     usage,
	}, nil
}

func (s *Service) GetModels(ctx context.Context) ([]string, error) {
	if s == nil || s.llm == nil {
		return nil, fmt.Errorf("pool не инициализирован")
	}

	return s.llm.GetModels(ctx)
}

func (s *Service) Embed(ctx context.Context, text string) ([]float32, error) {
	if s == nil || s.llm == nil {
		return nil, fmt.Errorf("pool не инициализирован")
	}

	return s.llm.Embed(ctx, text)
}

func mapPoolError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gendomain.ErrRunnerModelNotLoaded) {
		return err
	}

	if errors.Is(err, requestqueue.ErrQueueTimeout) {
		return err
	}

	return err
}
