package llmstream

import (
	"context"
	"strings"
	"testing"

	"github.com/magomedcoder/gen/pkg/chatstream"
	"github.com/magomedcoder/gen/pkg/domain"
)

func TestForwardLLMStreamChunks_textAndReasoning(t *testing.T) {
	ctx := context.Background()
	in := make(chan domain.LLMStreamChunk, 2)
	in <- domain.LLMStreamChunk{ReasoningContent: "think"}
	in <- domain.LLMStreamChunk{Content: "answer"}
	close(in)

	out := make(chan chatstream.ChatStreamChunk, 4)
	var acc strings.Builder
	go func() {
		ForwardLLMStreamChunks(ctx, out, 9, in, &acc)
		close(out)
	}()

	var kinds []chatstream.StreamChunkKind
	for ch := range out {
		kinds = append(kinds, ch.Kind)
		if ch.MessageID != 9 {
			t.Fatalf("id сообщения: %d", ch.MessageID)
		}
	}

	if acc.String() != "answer" {
		t.Fatalf("накопленный контент: %q", acc.String())
	}
	if len(kinds) != 2 || kinds[0] != chatstream.StreamChunkKindReasoning || kinds[1] != chatstream.StreamChunkKindText {
		t.Fatalf("виды: %v", kinds)
	}
}

func TestForwardLLMStreamChunks_respectsCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan domain.LLMStreamChunk)
	out := make(chan chatstream.ChatStreamChunk, 2)

	go func() {
		ForwardLLMStreamChunks(ctx, out, 1, in, &strings.Builder{})
		close(out)
	}()

	cancel()
	close(in)

	n := 0
	for range out {
		n++
	}
}
