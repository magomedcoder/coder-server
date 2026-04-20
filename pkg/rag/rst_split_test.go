package rag

import (
	"strings"
	"testing"
)

func TestRSTSetext_headingPath(t *testing.T) {
	doc := `Введение
========

Первый абзац введения.

Раздел A
--------

Текст раздела A.

Подраздел
~~~~~~~~~

Глубже вложенный текст.
`
	chunks := SplitText("guide.rst", doc, SplitOptions{ChunkSizeRunes: 500, ChunkOverlapRunes: 0})
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	var paths []string
	for _, c := range chunks {
		if hp, ok := c.Metadata["heading_path"].(string); ok && hp != "" {
			paths = append(paths, hp)
		}
	}

	if len(paths) == 0 {
		t.Fatal("expected at least one heading_path")
	}

	if !strings.Contains(strings.Join(paths, "|"), "Раздел A") {
		t.Fatalf("expected heading Раздел A in paths, got %v", paths)
	}
}

func TestRSTOverline_headingPath(t *testing.T) {
	doc := `
###########
Глава one
###########

Текст главы.

.. _anchor:

----------
Раздел two
----------

Конец.
`
	chunks := SplitText("book.rst", doc, SplitOptions{ChunkSizeRunes: 500, ChunkOverlapRunes: 0})
	var paths []string
	for _, c := range chunks {
		if hp, ok := c.Metadata["heading_path"].(string); ok && hp != "" {
			paths = append(paths, hp)
		}
	}

	joined := strings.Join(paths, "|")
	if !strings.Contains(joined, "Глава one") {
		t.Fatalf("expected overline chapter in paths, got %v", paths)
	}

	if !strings.Contains(joined, "Раздел two") {
		t.Fatalf("expected setext after skipped anchor, got %v", paths)
	}
}

func TestRSTExplicitMarkupSkipped(t *testing.T) {
	doc := `Параграф до.

.. note::

   Тело заметки продолжается здесь.

Ещё параграф.
`
	chunks := SplitText("n.rst", doc, SplitOptions{ChunkSizeRunes: 200, ChunkOverlapRunes: 0})
	var parts []string
	for _, c := range chunks {
		parts = append(parts, c.Text)
	}

	body := strings.Join(parts, "\n")
	if strings.Contains(body, ".. note::") {
		t.Fatalf("directive line should be stripped, got %q", body)
	}

	if !strings.Contains(body, "Тело заметки") {
		t.Fatalf("expected directive body kept, got %q", body)
	}

	if !strings.Contains(body, "Параграф до") || !strings.Contains(body, "Ещё параграф") {
		t.Fatalf("expected surrounding paragraphs, got %q", body)
	}
}
