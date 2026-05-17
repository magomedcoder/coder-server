package llmstream

import (
	"context"
	"strings"

	"github.com/magomedcoder/gen/pkg/chatstream"
	"github.com/magomedcoder/gen/pkg/domain"
)

func ForwardLLMStreamChunks(
	ctx context.Context,
	out chan<- chatstream.ChatStreamChunk,
	messageID int64,
	in <-chan domain.LLMStreamChunk,
	intoContent *strings.Builder,
) {
	for chunk := range in {
		if chunk.ReasoningContent != "" {
			select {
			case <-ctx.Done():
				return
			case out <- chatstream.ChatStreamChunk{
				Kind:      chatstream.StreamChunkKindReasoning,
				Text:      chunk.ReasoningContent,
				MessageID: messageID,
			}:
			}
		}

		if chunk.Content != "" {
			intoContent.WriteString(chunk.Content)
			select {
			case <-ctx.Done():
				return
			case out <- chatstream.ChatStreamChunk{
				Kind:      chatstream.StreamChunkKindText,
				Text:      chunk.Content,
				MessageID: messageID,
			}:
			}
		}
	}
}
