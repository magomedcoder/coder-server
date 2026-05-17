package rag

import (
	"strings"
	"testing"

	"github.com/magomedcoder/gen/pkg/domain"
)

func TestBuildMessageWithRAG_deepMapPrefix(t *testing.T) {
	scored := []domain.ScoredDocumentRAGChunk{
		{
			DocumentRAGChunk: domain.DocumentRAGChunk{
				ChunkIndex: 0,
				Text:       "alpha",
			},
			Score: 0.9,
		},
	}

	out := BuildMessageWithRAG("f.pdf", "вопрос?", scored, 8000, "- пункт один\n- пункт два")
	if !strings.Contains(out, "map-шаг") {
		t.Fatalf("ожидалось вступление map-шага: %s", out)
	}

	if !strings.Contains(out, "пункт один") || !strings.Contains(out, "alpha") {
		t.Fatalf("ожидались сводка deep и тело фрагмента: %s", out)
	}
}

func TestBuildMessageWithRAG_noDeep(t *testing.T) {
	scored := []domain.ScoredDocumentRAGChunk{
		{
			DocumentRAGChunk: domain.DocumentRAGChunk{
				ChunkIndex: 1,
				Text:       "beta",
			},
			Score: 0.8,
		},
	}

	out := BuildMessageWithRAG("x.txt", "q", scored, 5000, "")
	if strings.Contains(out, "map-шаг") {
		t.Fatalf("не ожидался map-шаг: %s", out)
	}

	if !strings.Contains(out, "beta") {
		t.Fatalf("нет фрагмента: %s", out)
	}
}

func TestFormatSourceCitation(t *testing.T) {
	meta := map[string]any{
		"heading_path":   "Глава 1 > Раздел 2",
		"pdf_page_start": 3,
		"pdf_page_end":   5,
	}

	got := FormatSourceCitation(meta)
	if !strings.Contains(got, "раздел=«Глава 1 > Раздел 2»") {
		t.Fatalf("missing heading citation: %q", got)
	}

	if !strings.Contains(got, "стр.=3–5") {
		t.Fatalf("missing page citation: %q", got)
	}
}
