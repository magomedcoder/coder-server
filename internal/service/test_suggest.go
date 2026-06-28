package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	gendomain "github.com/magomedcoder/lmpkg/domain"
)

type TestSuggestService struct {
	llm            *LLMRunnerService
	timeoutSeconds int32
}

func NewTestSuggestService(llm *LLMRunnerService, timeoutSeconds int32) *TestSuggestService {
	return &TestSuggestService{
		llm:            llm,
		timeoutSeconds: timeoutSeconds,
	}
}

func (s *TestSuggestService) Suggest(ctx context.Context, req domain.TestSuggestRequest) (domain.TestSuggestResponse, error) {
	if s == nil || s.llm == nil {
		return domain.TestSuggestResponse{}, fmt.Errorf("test suggest service не инициализирован")
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		return domain.TestSuggestResponse{}, fmt.Errorf("source обязателен")
	}

	system := `You are a test authoring assistant.
Respond with a single JSON object only:
{"summary":string,"test_code":string}
Write minimal focused tests for the provided source. Use idiomatic test style for the language.`
	user := fmt.Sprintf("Path: %s\nLanguage: %s\nSource:\n%s", strings.TrimSpace(req.Path), strings.TrimSpace(req.Language), source)
	if errText := strings.TrimSpace(req.Error); errText != "" {
		user += "\n\nFailure output:\n" + errText
	}

	maxTokens := int32(1024)
	temp := float32(0.2)
	rf := "json_object"
	messages := mapper.RunnerMessages(system, []domain.ChatMessage{
		{
			Role:    "user",
			Content: user,
		},
	}, nil, nil, 8192, false, nil)
	result, err := s.llm.CollectMessage(ctx, messages, nil, s.timeoutSeconds, &gendomain.GenerationParams{
		MaxTokens:      &maxTokens,
		Temperature:    &temp,
		ResponseFormat: &gendomain.ResponseFormat{Type: rf},
	})
	if err != nil {
		return domain.TestSuggestResponse{}, err
	}

	raw := strings.TrimSpace(result.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var resp domain.TestSuggestResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return domain.TestSuggestResponse{
			Summary:  "сырой ответ модели",
			TestCode: result.Content,
		}, nil
	}
	return resp, nil
}
