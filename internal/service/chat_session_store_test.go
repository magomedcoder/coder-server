package service

import (
	"testing"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func TestChatSessionStoreMergeUsesStoredWhenClientShorter(t *testing.T) {
	st := NewChatSessionStore(50)
	st.Record("s1", []domain.ChatMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	})

	got := st.Merge("s1", []domain.ChatMessage{
		{
			Role: "user",
			Content: "next",
		},
	})
	if len(got) != 3 {
		t.Fatalf("len=%d want 3", len(got))
	}

	if got[2].Content != "next" {
		t.Fatalf("last=%q", got[2].Content)
	}
}

func TestChatSessionStoreResolveGeneratesID(t *testing.T) {
	st := NewChatSessionStore(50)
	id := st.ResolveSessionID(&domain.ChatRequest{})
	if id == "" {
		t.Fatal("expected generated session id")
	}
}

func TestChatSessionStoreResolveKeepsProvidedID(t *testing.T) {
	st := NewChatSessionStore(50)
	req := domain.ChatRequest{
		Session: &domain.ChatSession{
			SessionID: "chat-fixed",
		},
	}

	if got := st.ResolveSessionID(&req); got != "chat-fixed" {
		t.Fatalf("got %q", got)
	}
}
