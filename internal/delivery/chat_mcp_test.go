package delivery

import (
	"testing"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func TestChatSystemPromptWithoutMCP(t *testing.T) {
	h := &Handler{}
	req := domain.ChatRequest{
		System: strPtr("You are helpful."),
	}

	got := h.chatSystemPrompt(req, false)
	if got != "You are helpful." {
		t.Fatalf("got %q", got)
	}
}

func TestChatSystemPromptSkipsWhenMCPEnabledFalse(t *testing.T) {
	h := &Handler{}
	disabled := false
	req := domain.ChatRequest{
		System: strPtr("Assistant"),
		Session: &domain.ChatSession{
			MCPEnabled: &disabled,
		},
	}

	got := h.chatSystemPrompt(req, false)
	if got != "Assistant" {
		t.Fatalf("got %q", got)
	}
}

func strPtr(s string) *string {
	return &s
}
