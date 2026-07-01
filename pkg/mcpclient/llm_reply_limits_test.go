package mcpclient

import (
	"strings"
	"testing"
)

func TestTruncateLLMReplyNoOp(t *testing.T) {
	s := "hello мир"
	if got := TruncateLLMReply(s, 100); got != s {
		t.Fatalf("получено %q, ожидалось %q", got, s)
	}
}

func TestTruncateLLMReplyCutsRunes(t *testing.T) {
	s := strings.Repeat("Я", 50)
	got := TruncateLLMReply(s, 10)
	if !strings.Contains(got, "[Coder: ответ обрезан") {
		t.Fatal("ожидалось маркер")
	}

	if strings.Count(got, "Я") > 10 {
		t.Fatalf("слишком много рун: %q", got)
	}
}
