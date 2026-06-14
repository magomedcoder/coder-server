package contextbuilder

import (
	"strings"
	"testing"

	"github.com/magomedcoder/coder-server/pkg/context"
)

func TestBuilderPriority(t *testing.T) {
	b := New(200, true)
	out := b.Build("", nil, &context.ChatContext{
		Selection: &context.SelectionContext{Text: "selected code"},
		Snippets: []context.ContextSnippet{
			{
				Path:    "a.rs",
				Content: "mention file",
				Source:  "mention",
			},
		},
	})

	if out == "" {
		t.Fatal("expected non-empty context")
	}

	for _, part := range []string{"Selection", "selected code", "File: a.rs"} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in %q", part, out)
		}
	}
}

func TestBuilderTrimsToBudget(t *testing.T) {
	b := New(600, false)
	huge := strings.Repeat("x", 10000)
	out := b.Build("", nil, &context.ChatContext{
		Selection: &context.SelectionContext{Text: huge},
	})

	if out == "" {
		t.Fatal("expected trimmed output")
	}

	if !strings.Contains(out, "[truncated]") {
		t.Fatalf("expected truncation marker in %q", out)
	}
}
