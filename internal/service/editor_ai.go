package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/domain"
	"github.com/magomedcoder/coder-server/internal/mapper"
	"github.com/magomedcoder/coder-server/pkg/security"
	gendomain "github.com/magomedcoder/lmpkg/domain"
)

// const inlineCompleteSystemPromptRu = `Ты - движок inline-дополнения кода. Продолжи код в позиции <CURSOR>. Верни ТОЛЬКО текст для вставки после курсора. Без markdown-ограждений, без пояснений, без повторения уже существующего кода.`
const inlineCompleteSystemPrompt = `You are an inline code completion engine. Continue the code at <CURSOR>. Return ONLY the text to insert after the cursor. No markdown fences, no explanation, no repetition of existing code.`

// const editSelectionSystemPromptRu = `Ты - ассистент редактирования кода. Пользователь выделил фрагмент исходного кода и дал инструкцию. Верни ТОЛЬКО текст замены для выделения - без markdown-ограждений, без пояснений, без окружающего контекста.`
const editSelectionSystemPrompt = `You are a code editing assistant. The user selected a fragment of source code and gave an instruction. Return ONLY the replacement text for the selection - no markdown fences, no explanation, no surrounding context.`

func (s *ChatService) InlineComplete(ctx context.Context, req domain.CompleteRequest) (domain.CompleteResponse, error) {
	if s == nil || s.llm == nil {
		return domain.CompleteResponse{}, fmt.Errorf("chat service не инициализирован")
	}

	path := strings.TrimSpace(req.Path)
	if path == "" {
		return domain.CompleteResponse{}, fmt.Errorf("path обязателен")
	}
	if strings.TrimSpace(req.Prefix) == "" && strings.TrimSpace(req.Suffix) == "" {
		return domain.CompleteResponse{}, fmt.Errorf("prefix или suffix обязателен")
	}

	if err := scanEditorAIInput(s.cfg, req.Prefix, req.Suffix); err != nil {
		return domain.CompleteResponse{}, err
	}

	language := strings.TrimSpace(req.Language)
	user := fmt.Sprintf(
		"File: %s\nLanguage: %s\n\n<PREFIX>\n%s\n<CURSOR>\n<SUFFIX>\n%s\n",
		path,
		language,
		req.Prefix,
		req.Suffix,
	)

	var editor *domain.EditorContext
	if req.CursorLine != nil || req.CursorColumn != nil {
		editor = &domain.EditorContext{
			Path:     path,
			Language: language,
			Snippet:  req.Prefix,
		}

		if req.CursorLine != nil {
			line := *req.CursorLine
			editor.CursorLine = &line
		}

		if req.CursorColumn != nil {
			col := *req.CursorColumn
			editor.CursorColumn = &col
		}
	}

	gen := inlineCompleteGenParams(req.Generate, s.cfg)
	result, err := s.llm.CollectMessage(ctx, mapper.RunnerMessages(
		inlineCompleteSystemPrompt,
		[]domain.ChatMessage{{Role: "user", Content: user}},
		editor,
		nil,
		s.cfg.ContextTokenBudget(),
		s.cfg.ContextScanSecrets(),
		nil,
	), nil, inlineCompleteTimeout(s.cfg), gen)
	if err != nil {
		return domain.CompleteResponse{}, err
	}

	text := sanitizeInlineCompletion(result.Content)
	return domain.CompleteResponse{Text: text}, nil
}

func (s *ChatService) EditSelection(ctx context.Context, req domain.EditRequest) (domain.EditResponse, error) {
	if s == nil || s.llm == nil {
		return domain.EditResponse{}, fmt.Errorf("chat service не инициализирован")
	}

	path := strings.TrimSpace(req.Path)
	selection := strings.TrimSpace(req.Selection)
	instruction := strings.TrimSpace(req.Instruction)
	if path == "" {
		return domain.EditResponse{}, fmt.Errorf("path обязателен")
	}

	if selection == "" {
		return domain.EditResponse{}, fmt.Errorf("selection обязателен")
	}

	if instruction == "" {
		return domain.EditResponse{}, fmt.Errorf("instruction обязателен")
	}

	if err := scanEditorAIInput(s.cfg, selection, instruction); err != nil {
		return domain.EditResponse{}, err
	}

	language := strings.TrimSpace(req.Language)
	user := fmt.Sprintf("File: %s\nLanguage: %s\nInstruction: %s\n\n<SELECTION>\n%s\n</SELECTION>\n", path, language, instruction, req.Selection)

	editor := &domain.EditorContext{
		Path:     path,
		Language: language,
		Snippet:  req.Selection,
	}

	if req.SelectionStartLine != nil {
		line := *req.SelectionStartLine
		editor.CursorLine = &line
	}

	if req.SelectionStartCol != nil {
		col := *req.SelectionStartCol
		editor.CursorColumn = &col
	}

	gen := editSelectionGenParams(req.Generate, s.cfg)
	result, err := s.llm.CollectMessage(ctx, mapper.RunnerMessages(
		editSelectionSystemPrompt,
		[]domain.ChatMessage{{Role: "user", Content: user}},
		editor,
		nil,
		s.cfg.ContextTokenBudget(),
		s.cfg.ContextScanSecrets(),
		nil,
	), nil, editSelectionTimeout(s.cfg), gen)
	if err != nil {
		return domain.EditResponse{}, err
	}

	text := sanitizeEditReplacement(result.Content)

	return domain.EditResponse{
		Text: text,
	}, nil
}

func scanEditorAIInput(cfg *config.Config, parts ...string) error {
	if cfg != nil && cfg.ModerationEnabled() {
		if security.ScanMessages(parts) {
			return fmt.Errorf("запрос отклонён moderation layer")
		}
	}

	if cfg != nil && cfg.ContextScanSecrets() {
		for _, part := range parts {
			if security.ContainsSecrets(part) {
				return fmt.Errorf("обнаружены секреты в запросе")
			}
		}
	}

	return nil
}

func inlineCompleteGenParams(in *domain.GenerateParams, cfg *config.Config) *gendomain.GenerationParams {
	gen := mapper.GenerateParams(in, cfg.Chat.Generate)
	maxTokens := int32(96)
	temp := float32(0.1)
	if in != nil && in.MaxTokens != nil && *in.MaxTokens > 0 {
		maxTokens = int32(*in.MaxTokens)
	}

	if in != nil && in.Temperature != nil && *in.Temperature > 0 {
		temp = float32(*in.Temperature)
	}

	gen.MaxTokens = &maxTokens
	gen.Temperature = &temp
	return gen
}

func editSelectionGenParams(in *domain.GenerateParams, cfg *config.Config) *gendomain.GenerationParams {
	gen := mapper.GenerateParams(in, cfg.Chat.Generate)
	maxTokens := int32(1024)
	temp := float32(0.2)
	if in != nil && in.MaxTokens != nil && *in.MaxTokens > 0 {
		maxTokens = int32(*in.MaxTokens)
	} else if gen.MaxTokens != nil {
		maxTokens = *gen.MaxTokens
	}

	if in != nil && in.Temperature != nil && *in.Temperature > 0 {
		temp = float32(*in.Temperature)
	} else if gen.Temperature != nil {
		temp = *gen.Temperature
	}

	gen.MaxTokens = &maxTokens
	gen.Temperature = &temp

	return gen
}

func inlineCompleteTimeout(cfg *config.Config) int32 {
	if cfg == nil {
		return 30
	}

	timeout := cfg.ChatTimeoutSeconds()
	if timeout > 60 {
		return 60
	}

	if timeout < 10 {
		return 10
	}

	return timeout
}

func editSelectionTimeout(cfg *config.Config) int32 {
	if cfg == nil {
		return 120
	}

	return cfg.ChatTimeoutSeconds()
}

func sanitizeInlineCompletion(text string) string {
	s := strings.TrimSpace(text)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 2 {
			s = strings.Join(lines[1:len(lines)-1], "\n")
		} else {
			return ""
		}
	}

	if strings.HasPrefix(s, "<SUFFIX>") || strings.HasPrefix(s, "<PREFIX>") {
		return ""
	}

	s = strings.TrimSpace(s)
	if len(s) > 2048 {
		s = s[:2048]
	}

	return s
}

func sanitizeEditReplacement(text string) string {
	s := strings.TrimSpace(text)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) > 2 {
			s = strings.Join(lines[1:len(lines)-1], "\n")
		} else {
			return ""
		}
	}

	if strings.HasPrefix(s, "<SELECTION>") {
		return ""
	}

	return strings.TrimSpace(s)
}
