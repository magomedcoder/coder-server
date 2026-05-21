package embedbatch

import (
	"context"
	"errors"
	"testing"
)

type stubEmbed struct {
	failBatchLen int
	calls        int
}

func (s *stubEmbed) Embed(_ context.Context, text string) ([]float32, error) {
	s.calls++
	return []float32{float32(len(text))}, nil
}

func (s *stubEmbed) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	s.calls++
	if s.failBatchLen > 0 && len(texts) > s.failBatchLen {
		return nil, errors.New("batch too large")
	}

	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{float32(i)}
	}

	return out, nil
}

func TestTextsBatches_recursiveOnBatchError(t *testing.T) {
	ctx := context.Background()
	st := &stubEmbed{failBatchLen: 2}
	texts := []string{"a", "b", "c", "d"}
	vec, err := TextsBatches(ctx, st, texts, 4)
	if err != nil {
		t.Fatal(err)
	}

	if len(vec) != 4 {
		t.Fatalf("len %d", len(vec))
	}

	if st.calls < 3 {
		t.Fatalf("ожидалось несколько вызовов embed, получено %d", st.calls)
	}
}

func TestTextsRecursive_mismatchedBatchLengthSplits(t *testing.T) {
	ctx := context.Background()
	llm := &badCountEmbed{}
	vec, err := TextsRecursive(ctx, llm, []string{"x", "y"})
	if err != nil {
		t.Fatal(err)
	}

	if len(vec) != 2 {
		t.Fatalf("len %d", len(vec))
	}
}

type badCountEmbed struct{}

func (b *badCountEmbed) Embed(context.Context, string) ([]float32, error) {
	return []float32{1}, nil
}

func (b *badCountEmbed) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 2 {
		return [][]float32{{1}}, nil
	}

	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{1}
	}

	return out, nil
}
