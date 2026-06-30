package llmhistory

import (
	"strings"

	"github.com/magomedcoder/lmpkg/domain"
)

func FilterHistoryForLLM(messages []*domain.Message) []*domain.Message {
	if len(messages) == 0 {
		return nil
	}

	out := make([]*domain.Message, 0, len(messages))
	for _, m := range messages {
		if m == nil {
			continue
		}

		if m.Role == domain.MessageRoleAssistant && strings.TrimSpace(m.Content) == "" {
			if strings.TrimSpace(m.ToolCallsJSON) == "" {
				continue
			}
		}

		out = append(out, m)
	}

	return out
}
