package rag

import (
	"strings"
	"testing"
)

func TestSanitizeRewrittenQuery(t *testing.T) {
	got := SanitizeRewrittenQuery("  одна строка  ")
	if got != "одна строка" {
		t.Fatalf("получено %q", got)
	}

	got = SanitizeRewrittenQuery("первая строка\nвторая игнорируется")
	if got != "первая строка" {
		t.Fatalf("ожидалась первая строка, получено %q", got)
	}

	long := strings.Repeat("а", QueryRewriteMaxRunes+50)
	got = SanitizeRewrittenQuery(long)
	if len([]rune(got)) != QueryRewriteMaxRunes {
		t.Fatalf("лимит: %d", len([]rune(got)))
	}
}

func TestSanitizeHyDEPseudoDocument(t *testing.T) {
	got := SanitizeHyDEPseudoDocument("  гипотетический ответ  ")
	if got != "гипотетический ответ" {
		t.Fatalf("получено %q", got)
	}

	long := strings.Repeat("b", HyDEMaxRunes+10)
	got = SanitizeHyDEPseudoDocument(long)
	if len([]rune(got)) != HyDEMaxRunes {
		t.Fatalf("лимит: %d", len([]rune(got)))
	}
}
