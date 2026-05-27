package mapper

import (
	"strings"
	"time"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/contextbuilder"
	"github.com/magomedcoder/coder-server/internal/domain"
	gendomain "github.com/magomedcoder/gen/pkg/domain"
)

func RunnerMessages(
	system string,
	input []domain.ChatMessage,
	editor *domain.EditorContext,
	chatCtx *domain.ChatContext,
	tokenBudget int,
	scanSecrets bool,
	prefixCache *contextbuilder.PrefixCache,
) []*gendomain.Message {
	out := make([]*gendomain.Message, 0, len(input)+3)
	now := time.Now()

	if systemPrompt := strings.TrimSpace(system); systemPrompt != "" {
		out = append(out, &gendomain.Message{
			Content:   systemPrompt,
			Role:      gendomain.MessageRoleSystem,
			CreatedAt: now,
		})
	}

	builder := contextbuilder.New(tokenBudget, scanSecrets)
	var ctxPrompt string
	if prefixCache != nil {
		key := contextbuilder.PrefixCacheKey(system, editor, chatCtx, tokenBudget, scanSecrets)
		if cached, ok := prefixCache.Get(key); ok {
			ctxPrompt = cached
		} else {
			ctxPrompt = builder.Build(system, editor, chatCtx)
			if ctxPrompt != "" {
				prefixCache.Put(key, ctxPrompt)
			}
		}
	} else {
		ctxPrompt = builder.Build(system, editor, chatCtx)
	}

	if ctxPrompt != "" {
		out = append(out, &gendomain.Message{
			Content:   ctxPrompt,
			Role:      gendomain.MessageRoleSystem,
			CreatedAt: now,
		})
	}

	for _, msg := range input {
		out = append(out, &gendomain.Message{
			Content:   msg.Content,
			Role:      gendomain.FromProtoRole(msg.Role),
			CreatedAt: now,
		})
	}

	return out
}

func GenerateParams(in *domain.GenerateParams, defaults config.GenerateConfig) *gendomain.GenerationParams {
	out := &gendomain.GenerationParams{}

	maxTokens := defaults.MaxTokens
	if in != nil && in.MaxTokens != nil {
		maxTokens = *in.MaxTokens
	}

	if maxTokens > 0 {
		v := int32(maxTokens)
		out.MaxTokens = &v
	}

	temp := float32(defaults.Temperature)
	if in != nil && in.Temperature != nil {
		temp = float32(*in.Temperature)
	}

	if temp > 0 {
		out.Temperature = &temp
	}

	return out
}

func TokenUsage(u *gendomain.StreamTokenUsage) *domain.TokenUsage {
	if u == nil {
		return nil
	}

	return &domain.TokenUsage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
	}
}
