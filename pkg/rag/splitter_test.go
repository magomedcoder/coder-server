package rag

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSplitText_paragraphs(t *testing.T) {
	text := "Первый абзац.\n\nВторой абзац с текстом."
	chunks := SplitText("a.txt", text, SplitOptions{ChunkSizeRunes: 200, ChunkOverlapRunes: 0})
	if len(chunks) < 1 {
		t.Fatal("ожидалось чанки")
	}

	joined := strings.Join(chunkTexts(chunks), " ")
	if !strings.Contains(joined, "Первый") || !strings.Contains(joined, "Второй") {
		t.Fatalf("неожиданное склейка: %q", joined)
	}

	if chunks[0].Metadata["file_name"] != "a.txt" {
		t.Fatalf("метаданные: %+v", chunks[0].Metadata)
	}

	if chunks[0].Metadata["source_format"] != "plain" {
		t.Fatalf("ожидалось source_format plain: %+v", chunks[0].Metadata)
	}

	if _, ok := chunks[0].Metadata["total_chunks"]; !ok {
		t.Fatalf("ожидалось total_chunks: %+v", chunks[0].Metadata)
	}
}

func TestSplitText_smallChunkForcesSplit(t *testing.T) {
	s := strings.Repeat("а", 50)
	chunks := SplitText("", s, SplitOptions{ChunkSizeRunes: 20, ChunkOverlapRunes: 0})
	if len(chunks) < 2 {
		t.Fatalf("ожидалось несколько чанков, получено %d", len(chunks))
	}

	total := 0
	for _, c := range chunks {
		total += utf8.RuneCountInString(c.Text)
	}

	if total < 50 {
		t.Fatalf("потеряны руны: total=%d", total)
	}
}

func TestSplitText_markdownHeadingPath(t *testing.T) {
	text := "# Intro\n\nHello here.\n\n## Deep\n\nBody text under deep."
	chunks := SplitText("doc.md", text, SplitOptions{ChunkSizeRunes: 200, ChunkOverlapRunes: 0})
	if len(chunks) < 1 {
		t.Fatal("ожидалось чанки")
	}

	var sawIntro, sawDeep bool
	for _, c := range chunks {
		if c.Metadata["source_format"] != "markdown" {
			t.Fatalf("ожидалось markdown: %+v", c.Metadata)
		}

		hp, _ := c.Metadata["heading_path"].(string)
		if strings.Contains(hp, "Intro") {
			sawIntro = true
		}

		if strings.Contains(hp, "Deep") {
			sawDeep = true
		}
	}

	if !sawIntro || !sawDeep {
		t.Fatalf("ожидалось paths Intro и Deep в чанках, получено %d (sawIntro=%v sawDeep=%v)", len(chunks), sawIntro, sawDeep)
	}
}

func TestSplitText_overlap(t *testing.T) {
	text := strings.Repeat("word ", 100)
	chunks := SplitText("f.md", text, SplitOptions{ChunkSizeRunes: 80, ChunkOverlapRunes: 20})
	if len(chunks) < 2 {
		t.Fatalf("нужно 2+ чанка, получено %d", len(chunks))
	}

	if _, ok := chunks[1].Metadata["chunk_index"]; !ok {
		t.Fatalf("ожидалось chunk_index в метаданные: %+v", chunks[1].Metadata)
	}
}

func chunkTexts(ch []Chunk) []string {
	out := make([]string, len(ch))
	for i := range ch {
		out[i] = ch[i].Text
	}

	return out
}

func TestRAGRegressionMarkdownStable(t *testing.T) {
	text := "# Intro\n\nHello here.\n\n## Deep\n\nBody text under deep."
	chunks := SplitText("doc.md", text, SplitOptions{ChunkSizeRunes: 200, ChunkOverlapRunes: 0})
	if len(chunks) != 2 {
		t.Fatalf("ожидалось ровно 2 чанка (разные heading_path), получено %d", len(chunks))
	}

	nTot, _ := chunks[0].Metadata["total_chunks"].(int)
	if nTot != 2 {
		t.Fatalf("total_chunks: ожидалось 2, получено %v", chunks[0].Metadata["total_chunks"])
	}

	for i, c := range chunks {
		if _, ok := c.Metadata["chunk_index"]; !ok {
			t.Fatalf("чанк %d: отсутствует chunk_index", i)
		}

		if c.Metadata["source_format"] != "markdown" {
			t.Fatalf("чанк %d: source_format", i)
		}

		if _, ok := c.Metadata["heading_path"].(string); !ok {
			t.Fatalf("чанк %d: отсутствует heading_path", i)
		}
	}
}

func TestSplitTextWithPDFPageBounds_spansTwoPages(t *testing.T) {
	p1 := "aaaaa"
	p2 := "bbbbb"
	doc := p1 + "\n\n" + p2
	r1 := utf8.RuneCountInString(p1)
	r2 := utf8.RuneCountInString(p2)
	bounds := []int{0, r1, r1 + 2 + r2}

	chunks := SplitTextWithPDFPageBounds("x.pdf", doc, SplitOptions{ChunkSizeRunes: 200, ChunkOverlapRunes: 0}, bounds)
	if len(chunks) != 1 {
		t.Fatalf("ожидалось 1 чанк, получено %d", len(chunks))
	}

	ps, _ := chunks[0].Metadata["pdf_page_start"].(int)
	pe, _ := chunks[0].Metadata["pdf_page_end"].(int)
	if ps != 1 || pe != 2 {
		t.Fatalf("pdf_page_start/end: ожидалось 1–2, получено %d–%d метаданные=%+v", ps, pe, chunks[0].Metadata)
	}

	if chunks[0].Metadata["reading_order"] != "extracted_sequence" {
		t.Fatalf("ожидалось reading_order на pdf чанк")
	}
}

func TestSplitTextWithPDFPageBounds_singlePageFragment(t *testing.T) {
	p1 := strings.Repeat("а", 60)
	doc := p1
	r1 := utf8.RuneCountInString(p1)
	bounds := []int{0, r1}

	chunks := SplitTextWithPDFPageBounds("d.pdf", doc, SplitOptions{ChunkSizeRunes: 25, ChunkOverlapRunes: 0}, bounds)
	if len(chunks) < 2 {
		t.Fatalf("ожидалось несколько чанков, получено %d", len(chunks))
	}

	for i, c := range chunks {
		ps, _ := c.Metadata["pdf_page_start"].(int)
		pe, _ := c.Metadata["pdf_page_end"].(int)
		if ps != 1 || pe != 1 {
			t.Fatalf("чанк %d: ожидалось pages 1–1, получено %d–%d", i, ps, pe)
		}
	}
}

func TestRAGRegressionPlainMetadataKeys(t *testing.T) {
	text := "Абзац один.\n\nАбзац два."
	chunks := SplitText("note.txt", text, SplitOptions{ChunkSizeRunes: 80, ChunkOverlapRunes: 0})
	if len(chunks) < 1 {
		t.Fatal("чанки")
	}

	for i, c := range chunks {
		for _, key := range []string{"file_name", "source_format", "chunk_index", "total_chunks"} {
			if _, ok := c.Metadata[key]; !ok {
				t.Fatalf("чанк %d: отсутствует %q in метаданные", i, key)
			}
		}

		if c.Metadata["source_format"] != "plain" {
			t.Fatalf("чанк %d: source_format", i)
		}

		if hp, ok := c.Metadata["heading_path"].(string); ok && hp != "" {
			t.Fatalf("plain text не должно set heading_path, получено %q", hp)
		}
	}
}
