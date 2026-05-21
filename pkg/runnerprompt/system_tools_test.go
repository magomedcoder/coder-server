package runnerprompt

import (
	"strings"
	"testing"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/mcpprompt"
)

func TestEnrichSystemMessage_includesCatalogAndInvocation(t *testing.T) {
	msg := domain.NewMessage(1, "base", domain.MessageRoleSystem)
	tools := []domain.Tool{
		{
			Name:        "echo_tool",
			Description: "d",
		},
	}
	EnrichSystemMessage(msg, SystemToolsOptions{
		Tools: tools,
		MCPEntries: []mcpprompt.ServerEntry{
			{
				ID:      1,
				Name:    "srv",
				Enabled: true,
			},
		},
	})

	if !strings.Contains(msg.Content, "[Tools]") {
		t.Fatal("отсутствует каталог")
	}

	if !strings.Contains(msg.Content, "JSON-массив") {
		t.Fatal("отсутствует блок вызова")
	}

	if !strings.Contains(msg.Content, "[MCP]") {
		t.Fatal("отсутствуют подсказки mcp")
	}
}
