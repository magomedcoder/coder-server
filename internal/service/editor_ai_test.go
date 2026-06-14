package service

import "testing"

func TestSanitizeInlineCompletion(t *testing.T) {
	raw := "```rust\nlet x = 1;\n```"
	if got := sanitizeInlineCompletion(raw); got != "let x = 1;" {
		t.Fatalf("ожидалось содержимое без markdown-ограждения, получено %q", got)
	}

	if got := sanitizeInlineCompletion("<PREFIX>foo"); got != "" {
		t.Fatalf("ожидалась пустая строка для маркера PREFIX, получено %q", got)
	}
}

func TestSanitizeEditReplacement(t *testing.T) {
	raw := "```\nreturn 42;\n```"
	if got := sanitizeEditReplacement(raw); got != "return 42;" {
		t.Fatalf("ожидалось содержимое без markdown-ограждения, получено %q", got)
	}

	if got := sanitizeEditReplacement("<SELECTION>"); got != "" {
		t.Fatalf("ожидалась пустая строка для маркера SELECTION, получено %q", got)
	}
}
