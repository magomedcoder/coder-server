package rag

import (
	"strings"
	"testing"

	"github.com/magomedcoder/coder-server/pkg/domain"
)

func TestSelectMultiFileCandidates_prefersHigherScoresWithLimits(t *testing.T) {
	candidates := []MultiFileCandidate{
		{
			FileIndex: 0,
			Score:     0.95,
			Chunk: domain.ScoredDocumentRAGChunk{
				DocumentRAGChunk: domain.DocumentRAGChunk{
					Text: strings.Repeat("A", 40),
				},
				Score: 0.95,
			},
		},
		{
			FileIndex: 0,
			Score:     0.90,
			Chunk: domain.ScoredDocumentRAGChunk{
				DocumentRAGChunk: domain.DocumentRAGChunk{
					Text: strings.Repeat("B", 40),
				},
				Score: 0.90,
			},
		},
		{
			FileIndex: 0,
			Score:     0.10,
			Chunk: domain.ScoredDocumentRAGChunk{
				DocumentRAGChunk: domain.DocumentRAGChunk{
					Text: strings.Repeat("C", 40),
				},
				Score: 0.10,
			},
		},
		{
			FileIndex: 1,
			Score:     0.80,
			Chunk: domain.ScoredDocumentRAGChunk{
				DocumentRAGChunk: domain.DocumentRAGChunk{
					Text: strings.Repeat("D", 40),
				},
				Score: 0.80,
			},
		},
	}

	out := SelectMultiFileCandidates(candidates, 140, 120)
	if len(out[0]) == 0 {
		t.Fatal("ожидался top chunk для файла 0")
	}

	if len(out[1]) == 0 {
		t.Fatal("ожидался selected chunk для файла 1")
	}

	if len(out[0]) > MultiFilePerFileLimit {
		t.Fatalf("превышен лимит per-file: %d", len(out[0]))
	}

	if out[0][0].Score < out[1][0].Score {
		t.Fatalf("ожидался более высокий score, file0=%v file1=%v", out[0][0].Score, out[1][0].Score)
	}
}

func TestSelectMultiFileCandidates_respectsBudgetInConflictingMultiFileCase(t *testing.T) {
	var candidates []MultiFileCandidate
	for i := range 6 {
		candidates = append(candidates, MultiFileCandidate{
			FileIndex: i % 2,
			Score:     1.0 - float64(i)*0.05,
			Chunk: domain.ScoredDocumentRAGChunk{
				DocumentRAGChunk: domain.DocumentRAGChunk{
					Text: strings.Repeat(string(rune('a'+i)), 90),
				},
				Score: 1.0 - float64(i)*0.05,
			},
		})
	}

	selected := SelectMultiFileCandidates(candidates, 220, 120)
	totalSelected := 0
	for _, rows := range selected {
		totalSelected += len(rows)
	}

	if totalSelected == 0 {
		t.Fatal("ожидался минимум один выбранный фрагмент")
	}

	if totalSelected > 2 {
		t.Fatalf("ожидался выбор с ограничением бюджета, получено фрагментов %d", totalSelected)
	}
}
