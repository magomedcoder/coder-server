package prompt

import (
	"strings"
	"testing"

	"github.com/magomedcoder/lmpkg/domain"
)

func TestFormatContentForRunner_toolRole(t *testing.T) {
	m := domain.NewMessage(1, "result body", domain.MessageRoleTool)
	m.ToolCallID = "call_1"
	m.ToolName = "mcp_2_habc"
	got := FormatContentForRunner(m)
	if !strings.HasPrefix(got, "[call_id=call_1] ") || !strings.Contains(got, "[mcp_2_habc] ") {
		t.Fatalf("получено %q", got)
	}

	if !strings.HasSuffix(got, "result body") {
		t.Fatal("отсутствует тело")
	}
}

func TestFormatContentForRunner_assistantToolCalls(t *testing.T) {
	m := domain.NewMessage(1, "thinking", domain.MessageRoleAssistant)
	m.ToolCallsJSON = `[{"tool_name":"x","parameters":{}}]`
	got := FormatContentForRunner(m)
	if !strings.Contains(got, ToolCallsBlockPrefix) {
		t.Fatalf("получено %q", got)
	}
}

func TestPrepareMessagesForRunner_clearsToolFields(t *testing.T) {
	in := []*domain.Message{
		func() *domain.Message {
			m := domain.NewMessage(1, "t", domain.MessageRoleTool)
			m.ToolName = "n"
			return m
		}(),
	}

	out := PrepareMessagesForRunner(in)
	if out[0].ToolName != "" || out[0].Content == "t" {
		t.Fatalf("поля tool должны быть встроены: content=%q name=%q", out[0].Content, out[0].ToolName)
	}
}
