package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/llmrunner"
)

type LLMRunnerService struct {
	inner *llmrunner.LLMRunnerService
}

func NewLLMRunnerService(runnerAddr string) (*LLMRunnerService, error) {
	inner, err := llmrunner.NewLLMRunnerService(runnerAddr, "")
	if err != nil {
		return nil, err
	}

	return &LLMRunnerService{inner: inner}, nil
}

func (s *LLMRunnerService) Close() error {
	if s == nil || s.inner == nil {
		return nil
	}
	
	return s.inner.Close()
}

func (s *LLMRunnerService) CheckConnection(ctx context.Context) (bool, error) {
	return s.inner.CheckConnection(ctx)
}

func (s *LLMRunnerService) ModelReady(ctx context.Context) error {
	lm, err := s.inner.GetLoadedModel(ctx)
	if err != nil {
		return fmt.Errorf("GetLoadedModel: %w", err)
	}

	if lm == nil || !lm.GetLoaded() {
		return fmt.Errorf("модель не загружена на gen-runner")
	}
	return nil
}

func (s *LLMRunnerService) SendMessage(
	ctx context.Context,
	messages []*domain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *domain.GenerationParams,
) (chan domain.LLMStreamChunk, error) {
	return s.inner.SendMessage(ctx, messages, stopSequences, timeoutSeconds, genParams)
}

func (s *LLMRunnerService) CollectMessage(
	ctx context.Context,
	messages []*domain.Message,
	stopSequences []string,
	timeoutSeconds int32,
	genParams *domain.GenerationParams,
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
