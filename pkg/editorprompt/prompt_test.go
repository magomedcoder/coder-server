package editorprompt

import (
	"strings"
	"testing"
)

func TestWrapUserText(t *testing.T) {
	got := WrapUserText("hello")
	if !strings.Contains(got, "```\nhello\n```") {
		t.Fatalf("получено %q", got)
	}
}

func TestBuildSystemPrompt_fixPreservesMarkdown(t *testing.T) {
	got := BuildSystemPrompt(TransformFix, true)
	if !strings.Contains(got, "орфографию") || !strings.Contains(got, "Markdown") {
		t.Fatalf("получено %q", got)
	}
}
