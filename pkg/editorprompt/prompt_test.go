package editorprompt

import (
	"strings"
	"testing"

	"github.com/magomedcoder/gen/api/pb/editorpb"
)

func TestWrapUserText(t *testing.T) {
	got := WrapUserText("hello")
	if !strings.Contains(got, "```\nhello\n```") {
		t.Fatalf("получено %q", got)
	}
}

func TestBuildSystemPrompt_fixPreservesMarkdown(t *testing.T) {
	got := BuildSystemPrompt(editorpb.TransformType_TRANSFORM_TYPE_FIX, true)
	if !strings.Contains(got, "орфографию") || !strings.Contains(got, "Markdown") {
		t.Fatalf("получено %q", got)
	}
}
