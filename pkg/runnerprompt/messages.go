package runnerprompt

import (
	"fmt"
	"strings"

	"github.com/magomedcoder/gen/pkg/domain"
)

const (
	ToolCallsBlockPrefix = "\n[tool_calls]: "
	ToolCallsOnlyPrefix  = "[tool_calls]: "
)

func FormatContentForRunner(m *domain.Message) string {
	if m == nil {
		return ""
	}

	c := m.Content
	if m.Role == domain.MessageRoleTool {
		var b strings.Builder
		if id := strings.TrimSpace(m.ToolCallID); id != "" {
			fmt.Fprintf(&b, "[call_id=%s] ", id)
		}

		if name := strings.TrimSpace(m.ToolName); name != "" {
			fmt.Fprintf(&b, "[%s] ", name)
		}

		b.WriteString(c)
		return b.String()
	}

	if m.Role == domain.MessageRoleAssistant {
		tc := strings.TrimSpace(m.ToolCallsJSON)
		if tc == "" {
			return c
		}

		if strings.TrimSpace(c) != "" {
			return c + ToolCallsBlockPrefix + tc
		}

		return ToolCallsOnlyPrefix + tc
	}

	return c
}

func PrepareMessagesForRunner(messages []*domain.Message) []*domain.Message {
	if len(messages) == 0 {
		return nil
	}

	out := make([]*domain.Message, 0, len(messages))
	for _, m := range messages {
		if m == nil {
			continue
		}

		clone := *m
		clone.Content = FormatContentForRunner(m)
		clone.ToolCallID = ""
		clone.ToolName = ""
		clone.ToolCallsJSON = ""
		out = append(out, &clone)
	}

	return out
}
