package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/magomedcoder/gen/pkg/domain"
)

func makeRunnerMessages(system string, input []ChatMessage, editor *EditorContext) []*domain.Message {
	out := make([]*domain.Message, 0, len(input)+2)
	now := time.Now()

	if systemPrompt := strings.TrimSpace(system); systemPrompt != "" {
		out = append(out, &domain.Message{
			Content:   systemPrompt,
			Role:      domain.MessageRoleSystem,
			CreatedAt: now,
		})
	}

	if editorPrompt := buildEditorPrompt(editor); editorPrompt != "" {
		out = append(out, &domain.Message{
			Content:   editorPrompt,
			Role:      domain.MessageRoleSystem,
			CreatedAt: now,
		})
	}

	for _, msg := range input {
		out = append(out, &domain.Message{
			Content:   msg.Content,
			Role:      domain.FromProtoRole(msg.Role),
			CreatedAt: now,
		})
	}

	return out
}

func mapGenerateParams(in *GenerateParams) *domain.GenerationParams {
	if in == nil {
		return nil
	}

	out := &domain.GenerationParams{}
	if in.MaxTokens != nil {
		v := int32(*in.MaxTokens)
		out.MaxTokens = &v
	}

	if in.Temperature != nil {
		v := float32(*in.Temperature)
		out.Temperature = &v
	}

	return out
}

func buildEditorPrompt(editor *EditorContext) string {
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
