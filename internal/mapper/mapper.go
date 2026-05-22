package mapper

import (
	"fmt"
	"github.com/magomedcoder/coder-server/internal/config"
	"strings"
	"time"

	gendomain "github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/coder-server/internal/domain"
)

func RunnerMessages(system string, input []domain.ChatMessage, editor *domain.EditorContext) []*gendomain.Message {
	out := make([]*gendomain.Message, 0, len(input)+2)
	now := time.Now()

	if systemPrompt := strings.TrimSpace(system); systemPrompt != "" {
		out = append(out, &gendomain.Message{
			Content:   systemPrompt,
			Role:      gendomain.MessageRoleSystem,
			CreatedAt: now,
		})
	}

	if editorPrompt := EditorPrompt(editor); editorPrompt != "" {
		out = append(out, &gendomain.Message{
			Content:   editorPrompt,
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

func EditorPrompt(editor *domain.EditorContext) string {
	if editor == nil {
		return ""
	}

	parts := make([]string, 0, 5)
	if p := strings.TrimSpace(editor.Path); p != "" {
		parts = append(parts, "path: "+p)
	}

	if l := strings.TrimSpace(editor.Language); l != "" {
		parts = append(parts, "language: "+l)
	}

	if editor.CursorLine != nil && editor.CursorColumn != nil {
		parts = append(parts, fmt.Sprintf("cursor: %d:%d", *editor.CursorLine, *editor.CursorColumn))
	}

	if s := strings.TrimSpace(editor.Snippet); s != "" {
		parts = append(parts, "snippet:\n"+s)
	}

	if len(parts) == 0 {
		return ""
	}

	return "Editor context:\n" + strings.Join(parts, "\n")
}
