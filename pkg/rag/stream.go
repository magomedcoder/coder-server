package rag

import (
	"context"
	"strings"

	"github.com/magomedcoder/coder-server/pkg/domain"
)

func DrainStreamContent(ctx context.Context, ch <-chan domain.LLMStreamChunk) (string, error) {
	var b strings.Builder
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				return b.String(), nil
			}
			b.WriteString(chunk.Content)
		}
	}
}
