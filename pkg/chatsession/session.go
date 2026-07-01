package chatsession

import (
	"encoding/json"
	"strings"

	"github.com/magomedcoder/coder-server/pkg/domain"
)

func NormalizeWebSearchProvider(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "", "brave", "google", "yandex", "multi":
		return s
	default:
		return ""
	}
}

func GenParamsFromSettings(settings *domain.ChatSessionSettings) (stopSequences []string, timeoutSeconds int32, genParams *domain.GenerationParams) {
	if settings == nil {
		return nil, 0, nil
	}

	if len(settings.StopSequences) > 0 {
		stopSequences = append([]string(nil), settings.StopSequences...)
	}

	timeoutSeconds = settings.TimeoutSeconds
	et := settings.ModelReasoningEnabled
	genParams = &domain.GenerationParams{
		Temperature:    settings.Temperature,
		TopK:           settings.TopK,
		TopP:           settings.TopP,
		EnableThinking: &et,
	}

	if settings.JSONMode {
		jsonSchema := strings.TrimSpace(settings.JSONSchema)
		var schemaPtr *string
		if jsonSchema != "" {
			schemaPtr = &jsonSchema
		}

		genParams.ResponseFormat = &domain.ResponseFormat{
			Type:   "json_object",
			Schema: schemaPtr,
		}
	}

	if parsedTools := ParseToolsJSON(settings.ToolsJSON); len(parsedTools) > 0 {
		genParams.Tools = parsedTools
	}

	return stopSequences, timeoutSeconds, genParams
}

func ParseToolsJSON(raw string) []domain.Tool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	var tools []domain.Tool
	if err := json.Unmarshal([]byte(trimmed), &tools); err != nil {
		return nil
	}

	return tools
}

func NormalizeWebSearchMaxResults(n int) int {
	if n <= 0 {
		return 20
	}

	if n > 50 {
		return 50
	}

	return n
}
