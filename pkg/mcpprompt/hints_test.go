package mcpprompt

import (
	"strings"
	"testing"

	"github.com/magomedcoder/gen/pkg/domain"
)

func TestBuildSessionHints_containsServerAndAliasGuidance(t *testing.T) {
	got := BuildSessionHints([]ServerEntry{
		{ID: 9, Name: "Demo", Enabled: true},
	})
	if !strings.Contains(got, "id=9 · Demo") {
		t.Fatalf("missing server line: %q", got)
	}
	if !strings.Contains(got, "mcp_<id>_h<hex>") {
		t.Fatalf("missing alias guidance: %q", got)
	}
	if strings.Contains(got, "gen_mcp_") {
		t.Fatalf("must not mention gen_mcp_*: %q", got)
	}
}

func TestBuildSessionHints_empty(t *testing.T) {
	if BuildSessionHints(nil) != "" {
		t.Fatal("expected empty for nil entries")
	}
}

func TestBuildToolCatalog(t *testing.T) {
	if got := BuildToolCatalog(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	got := BuildToolCatalog([]domain.Tool{{Name: "web_search"}})
	if !strings.Contains(got, "web_search") {
		t.Fatalf("missing tool name: %q", got)
	}
}

func TestAppendBlock(t *testing.T) {
	if got := AppendBlock("base", "extra"); got != "base\n\nextra" {
		t.Fatalf("got %q", got)
	}
	if got := AppendBlock("base", ""); got != "base" {
		t.Fatalf("got %q", got)
	}
}
