package chatstream

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/magomedcoder/coder-server/pkg/domain"
)

func TestBuildStreamMetaFullDocument_excerpt(t *testing.T) {
	m, err := BuildStreamMetaFullDocument(7, "  hello очень длинный текст для превью  ")
	if err != nil {
		t.Fatal(err)
	}

	var w map[string]any
	if err := json.Unmarshal([]byte(m.SourcesJSON), &w); err != nil {
		t.Fatal(err)
	}

	if w["mode"] != ragModeFullDocument || int(w["file_id"].(float64)) != 7 {
		t.Fatalf("wire-структура: %+v", w)
	}

	excerpt, _ := w["full_document_excerpt"].(string)
	if !strings.Contains(excerpt, "hello") {
		t.Fatalf("выдержка: %q", excerpt)
	}

	if m.Sources == nil || m.Sources.FileID != 7 || m.Sources.Mode != ragModeFullDocument {
		t.Fatalf("типизированный payload: %+v", m.Sources)
	}
}

func TestBuildStreamMetaVector_modes(t *testing.T) {
	scored := []domain.ScoredDocumentRAGChunk{
		{
			DocumentRAGChunk: domain.DocumentRAGChunk{
				ChunkIndex: 1,
				Text:       "alpha beta gamma delta preview body",
				Metadata: map[string]any{
					"heading_path":   "Intro › A",
					"pdf_page_start": float64(2),
					"pdf_page_end":   float64(2),
				},
			},
			Score: 0.88,
		},
	}

	m, err := BuildStreamMetaVector(42, 5, 2, scored, 0, false, 1)
	if err != nil {
		t.Fatal(err)
	}

	if m.Mode != ragModeVectorRAG {
		t.Fatalf("режим: %s", m.Mode)
	}

	var w map[string]any
	if err := json.Unmarshal([]byte(m.SourcesJSON), &w); err != nil {
		t.Fatal(err)
	}

	if int(w["file_id"].(float64)) != 42 || int(w["top_k"].(float64)) != 5 {
		t.Fatalf("wire-структура: %+v", w)
	}

	m2, err := BuildStreamMetaVector(1, 3, 0, scored, 2, true, 0)
	if err != nil {
		t.Fatal(err)
	}

	if m2.Mode != ragModeVectorRAGDeep {
		t.Fatalf("режим deep: %s", m2.Mode)
	}
}
