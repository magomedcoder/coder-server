package prompt

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
	runnertemplate "github.com/magomedcoder/gen-runner/template"
)

func BuildChatPrompt(chatTemplateJinja string, messages []*domain.Message) (string, error) {
	j := strings.TrimSpace(chatTemplateJinja)
	if j == "" {
		return "", fmt.Errorf("runnerprompt: пустой chat_template")
	}

	preset, err := runnertemplate.Named(j)
	if err != nil {
		return "", err
	}

	return RenderMatchedPreset(preset, messages)
}

func RenderMatchedPreset(preset *runnertemplate.MatchedPreset, messages []*domain.Message) (string, error) {
	if preset == nil {
		return "", fmt.Errorf("runnerprompt: пресет не задан")
	}

	raw, err := io.ReadAll(preset.Reader())
	if err != nil {
		return "", err
	}

	tmpl, err := runnertemplate.Parse(strings.TrimSpace(string(raw)))
	if err != nil {
		return "", fmt.Errorf("runnerprompt: пресет %q: %w", preset.Name, err)
	}

	msgs, err := messagesForTemplate(messages)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, runnertemplate.Values{Messages: msgs}); err != nil {
		return "", fmt.Errorf("runnerprompt: пресет %q: %w", preset.Name, err)
	}

	return buf.String(), nil
}

func PresetStopSequences(chatTemplateJinja string) ([]string, error) {
	j := strings.TrimSpace(chatTemplateJinja)
	if j == "" {
		return nil, nil
	}

	preset, err := runnertemplate.Named(j)
	if err != nil {
		return nil, err
	}

	return runnertemplate.PresetStopSequences(preset), nil
}

func messagesForTemplate(messages []*domain.Message) ([]runnertemplate.Message, error) {
	out := make([]runnertemplate.Message, 0, len(messages))
	for _, m := range messages {
		if m == nil {
			continue
		}

		tm, err := messageToTemplate(m)
		if err != nil {
			return nil, err
		}

		out = append(out, tm)
	}

	return out, nil
}

func messageToTemplate(m *domain.Message) (runnertemplate.Message, error) {
	tm := runnertemplate.Message{
		Role:    string(m.Role),
		Content: m.Content,
	}

	switch m.Role {
	case domain.MessageRoleAssistant:
		body, tcJSON := splitAssistantToolCalls(m.Content)
		if tcJSON == "" {
			tcJSON = strings.TrimSpace(m.ToolCallsJSON)
		}

		tm.Content = body
		if tcJSON != "" {
			calls, err := parseToolCallsJSON(tcJSON)
			if err != nil {
				return runnertemplate.Message{}, fmt.Errorf("разбор tool_calls: %w", err)
			}

			tm.ToolCalls = calls
		}
	case domain.MessageRoleTool:
		body, callID, toolName := parseToolRoleContent(m.Content)
		if callID == "" {
			callID = strings.TrimSpace(m.ToolCallID)
		}

		if toolName == "" {
			toolName = strings.TrimSpace(m.ToolName)
		}

		tm.Content = body
		tm.ToolCallID = callID
		tm.ToolName = toolName
	}

	return tm, nil
}

func splitAssistantToolCalls(content string) (body, toolCallsJSON string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", ""
	}

	if i := strings.LastIndex(content, ToolCallsBlockPrefix); i >= 0 {
		return strings.TrimSpace(content[:i]), strings.TrimSpace(content[i+len(ToolCallsBlockPrefix):])
	}

	if strings.HasPrefix(content, ToolCallsOnlyPrefix) {
		return "", strings.TrimSpace(content[len(ToolCallsOnlyPrefix):])
	}

	return content, ""
}

func parseToolRoleContent(content string) (body, callID, toolName string) {
	rest := strings.TrimSpace(content)
	if rest == "" {
		return "", "", ""
	}

	if strings.HasPrefix(rest, "[call_id=") {
		end := strings.Index(rest, "]")
		if end > len("[call_id=") {
			callID = strings.TrimSpace(rest[len("[call_id="):end])
			rest = strings.TrimSpace(rest[end+1:])
		}
	}

	if strings.HasPrefix(rest, "[") {
		end := strings.Index(rest, "]")
		if end > 1 {
			toolName = strings.TrimSpace(rest[1:end])
			rest = strings.TrimSpace(rest[end+1:])
		}
	}

	return rest, callID, toolName
}
