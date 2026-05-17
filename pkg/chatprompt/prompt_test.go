package chatprompt

import (
	"strings"
	"testing"

	"github.com/magomedcoder/gen/pkg/domain"
	"github.com/magomedcoder/gen/pkg/llmhistory"
)

const testMaxContextTokens = 512

func TestAssemblePromptMessages_keepsInstructionSeparateFromDocumentContext(t *testing.T) {
	sessionID := int64(11)
	systemPolicy := domain.NewMessage(sessionID, "system policy", domain.MessageRoleSystem)
	history := []*domain.Message{
		domain.NewMessage(sessionID, "previous assistant", domain.MessageRoleAssistant),
	}

	userInstruction := domain.NewMessage(sessionID, "answer in 3 bullets", domain.MessageRoleUser)
	blocks := []DocumentContextBlock{
		{Title: "Файл: notes.txt", Body: "```txt\nfacts\n```"},
	}

	out := AssemblePromptMessages(sessionID, systemPolicy, history, userInstruction, blocks)
	if len(out) != 4 {
		t.Fatalf("unexpected message count: %d", len(out))
	}

	if out[0].Role != domain.MessageRoleSystem || out[0].Content != "system policy" {
		t.Fatalf("first message must be system policy, got role=%s content=%q", out[0].Role, out[0].Content)
	}

	if out[1].Role != domain.MessageRoleAssistant {
		t.Fatalf("second message must be history assistant, got role=%s", out[1].Role)
	}

	if out[2].Role != domain.MessageRoleSystem {
		t.Fatalf("third message must be document context system message, got role=%s", out[2].Role)
	}

	if !strings.Contains(out[2].Content, DocumentContextHierarchyInstruction) {
		t.Fatalf("document context block must include hierarchy instruction, got %q", out[2].Content)
	}

	if out[3].Role != domain.MessageRoleUser || out[3].Content != "answer in 3 bullets" {
		t.Fatalf("last message must be raw user instruction, got role=%s content=%q", out[3].Role, out[3].Content)
	}
}

func TestAssemblePromptMessages_withoutDocumentContext(t *testing.T) {
	sessionID := int64(12)
	systemPolicy := domain.NewMessage(sessionID, "sys", domain.MessageRoleSystem)
	history := []*domain.Message{
		domain.NewMessage(sessionID, "prev user", domain.MessageRoleUser),
	}
	userInstruction := domain.NewMessage(sessionID, "latest user request", domain.MessageRoleUser)

	out := AssemblePromptMessages(sessionID, systemPolicy, history, userInstruction, nil)
	if len(out) != 3 {
		t.Fatalf("unexpected message count: %d", len(out))
	}

	if out[0] != systemPolicy {
		t.Fatal("system policy must stay first")
	}

	if out[1] != history[0] {
		t.Fatal("history message must be preserved")
	}

	if out[2] != userInstruction {
		t.Fatal("user instruction must stay last")
	}
}

func TestFormatDocumentContextBlock(t *testing.T) {
	got := FormatDocumentContextBlock("Файл: a.txt", "```txt\nbody\n```")
	if !strings.Contains(got, "### Файл: a.txt") {
		t.Fatalf("missing heading: %q", got)
	}

	if !strings.Contains(got, "```txt\nbody\n```") {
		t.Fatalf("missing body: %q", got)
	}
}

func TestApplyInstructionSafeBudgetManager_dropsDocumentContextFirst(t *testing.T) {
	maxTok := llmhistory.NormalizeApproxMaxTokens(testMaxContextTokens)
	systemPolicy := domain.NewMessage(1, "system", domain.MessageRoleSystem)
	history := []*domain.Message{
		domain.NewMessage(1, "history", domain.MessageRoleAssistant),
	}

	userInstruction := domain.NewMessage(1, "latest user instruction", domain.MessageRoleUser)
	blocks := []DocumentContextBlock{
		{
			Title:      "RAG-контекст: big.txt",
			Body:       strings.Repeat("A", 800),
			SourceType: "rag",
			SourceFile: "big.txt",
		},
	}

	out, metrics := ApplyInstructionSafeBudgetManager(maxTok, systemPolicy, history, userInstruction, blocks)
	if len(out) != 0 {
		t.Fatalf("expected context to be dropped first, got %d blocks", len(out))
	}

	if metrics.DroppedRunesTotal == 0 {
		t.Fatal("expected dropped runes metrics")
	}

	if metrics.DroppedRunesByFile["big.txt"] == 0 {
		t.Fatalf("expected by-file metric for big.txt, got %#v", metrics.DroppedRunesByFile)
	}

	if metrics.DroppedRunesBySource["rag"] == 0 {
		t.Fatalf("expected by-source metric for rag, got %#v", metrics.DroppedRunesBySource)
	}
}

func TestApplyInstructionSafeBudgetManager_keepsStrictFormatInstructionWithLongDocument(t *testing.T) {
	maxTok := llmhistory.NormalizeApproxMaxTokens(testMaxContextTokens)
	systemPolicy := domain.NewMessage(1, "system policy", domain.MessageRoleSystem)
	history := []*domain.Message{
		domain.NewMessage(1, "history", domain.MessageRoleAssistant),
	}

	strictInstruction := `Ответь строго JSON-объектом {"status":"ok","items":[]}.`
	userInstruction := domain.NewMessage(1, strictInstruction, domain.MessageRoleUser)
	blocks := []DocumentContextBlock{
		{
			Title:      "RAG-контекст: long.txt",
			Body:       strings.Repeat("L", 4000),
			SourceType: "rag",
			SourceFile: "long.txt",
		},
	}

	trimmedBlocks, _ := ApplyInstructionSafeBudgetManager(maxTok, systemPolicy, history, userInstruction, blocks)
	messages := AssemblePromptMessages(1, systemPolicy, history, userInstruction, trimmedBlocks)
	last := messages[len(messages)-1]
	if last.Role != domain.MessageRoleUser || last.Content != strictInstruction {
		t.Fatalf("strict user instruction must stay unchanged, got role=%s content=%q", last.Role, last.Content)
	}
}
