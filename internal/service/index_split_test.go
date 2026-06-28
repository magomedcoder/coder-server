package service

import (
	"strings"
	"testing"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func TestExpandOversizedChunks(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 500; i++ {
		b.WriteString(strings.Repeat("x", 40))
		b.WriteByte('\n')
	}

	big := b.String()
	chunks := expandOversizedChunks([]domain.IndexChunk{{
		ID:       "chunk-src-lib.rs",
		Path:     "src/lib.rs",
		Language: "rust",
		Content:  big,
		Symbol:   "lib",
	}})
	if len(chunks) < 2 {
		t.Fatalf("ожидалось несколько сегментов, получено %d", len(chunks))
	}

	if chunks[0].ID != "chunk-src-lib.rs" {
		t.Fatalf("первый идентификатор\n=%q", chunks[0].ID)
	}

	if chunks[1].ID != "chunk-src-lib.rs-1" {
		t.Fatalf("второй идентификатор\n=%q", chunks[1].ID)
	}

	if chunks[0].Symbol != "lib" {
		t.Fatalf("symbol=%q", chunks[0].Symbol)
	}
}

func TestExpandOversizedChunksSmallUnchanged(t *testing.T) {
	in := []domain.IndexChunk{
		{
			ID:      "a",
			Content: "fn main() {}",
		},
	}
	out := expandOversizedChunks(in)
	if len(out) != 1 || out[0].Content != in[0].Content {
		t.Fatalf("unexpected %+v", out)
	}
}
