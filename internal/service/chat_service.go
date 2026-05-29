package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/contextbuilder"
	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
)

type ChatService struct {
	cfg         *config.Config
	llm         *LLMRunnerService
	index       *RepoIndex
	prefixCache *contextbuilder.PrefixCache
}

func NewChatService(cfg *config.Config, llm *LLMRunnerService, index *RepoIndex, prefixCache *contextbuilder.PrefixCache) *ChatService {
	return &ChatService{
		cfg:         cfg,
		llm:         llm,
		index:       index,
		prefixCache: prefixCache,
	}
}

func (s *ChatService) Complete(ctx context.Context, req domain.ChatRequest) (domain.ChatResponse, error) {
	if s == nil || s.llm == nil || req.System == nil {
		return domain.ChatResponse{}, fmt.Errorf("chat service не инициализирован")
	}

	EnrichContextFromSearch(ctx, s.index, s.llm, &req)
	req.Messages = trimMessages(req.Messages, s.cfg.HistoryMaxMessages())

	messages := mapper.RunnerMessages(*req.System, req.Messages, req.Editor, req.Context, s.cfg.ContextTokenBudget(), s.cfg.ContextScanSecrets(), s.prefixCache)
	genParams := mergeGenerateParams(req, s.cfg)
	timeout := chatTimeout(req, s.cfg)

	result, err := s.llm.CollectMessage(ctx, messages, stopSequences(req), timeout, genParams)
	if err != nil {
		return domain.ChatResponse{}, err
	}

	return domain.ChatResponse{
		Message: domain.ChatMessage{Role: "assistant", Content: result.Content},
		Finish:  "stop",
		Usage:   mapper.TokenUsage(result.Usage),
	}, nil
}

func EnrichContextFromSearch(ctx context.Context, index *RepoIndex, llm *LLMRunnerService, req *domain.ChatRequest) {
	if req == nil || req.Search == nil || index == nil {
		return
	}

	ws := strings.TrimSpace(req.Search.WorkspaceID)
	if ws == "" {
		return
	}

	query := strings.TrimSpace(req.Search.Query)
	if query == "" && len(req.Messages) > 0 {
		query = strings.TrimSpace(req.Messages[len(req.Messages)-1].Content)
	}

	if query == "" {
		return
	}

	limit := req.Search.Limit
	if limit <= 0 {
		limit = 5
	}

	mode := strings.TrimSpace(req.Search.Mode)
	if mode == "" {
		mode = "hybrid"
	}

	resp, err := index.Search(ctx, llm, domain.SearchRequest{
		WorkspaceID: ws,
		Query:       query,
		Limit:       limit,
		Mode:        mode,
	})
	if err != nil || len(resp.Hits) == 0 {
		return
	}

	if req.Context == nil {
		req.Context = &domain.ChatContext{}
	}

	for _, hit := range resp.Hits {
		content := hit.Snippet
		if hit.Symbol != "" {
			content = "// " + hit.Symbol + "\n" + hit.Snippet
		}

		req.Context.Snippets = append(req.Context.Snippets, domain.ContextSnippet{
			Path:     hit.Path,
			Language: hit.Language,
			Content:  content,
			Source:   "codebase",
		})
	}
}

func trimMessages(messages []domain.ChatMessage, max int) []domain.ChatMessage {
	if max <= 0 || len(messages) <= max {
		return messages
	}

	return messages[len(messages)-max:]
}

func stopSequences(req domain.ChatRequest) []string {
	if req.Session == nil {
		return nil
	}

	return req.Session.StopSequences
}

func chatTimeout(req domain.ChatRequest, cfg *config.Config) int32 {
	if req.Session != nil && req.Session.TimeoutSeconds != nil && *req.Session.TimeoutSeconds > 0 {
		return int32(*req.Session.TimeoutSeconds)
	}

	return cfg.ChatTimeoutSeconds()
}

func mergeGenerateParams(req domain.ChatRequest, cfg *config.Config) *gendomain.GenerationParams {
	gen := mapper.GenerateParams(req.Generate, cfg.Chat.Generate)
	if req.Session == nil {
		return gen
	}

	if req.Session.Temperature != nil {
		temp := float32(*req.Session.Temperature)
		gen.Temperature = &temp
	}

	return gen
}
