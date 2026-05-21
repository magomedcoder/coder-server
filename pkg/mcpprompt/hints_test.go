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
		t.Fatalf("отсутствует строка сервера: %q", got)
	}
	if !strings.Contains(got, "mcp_<id>_h<hex>") {
		t.Fatalf("отсутствует подсказка alias: %q", got)
	}
	if strings.Contains(got, "gen_mcp_") {
		t.Fatalf("не должно быть gen_mcp_*: %q", got)
	}
}

func TestBuildSessionHints_empty(t *testing.T) {
	if BuildSessionHints(nil) != "" {
		t.Fatal("ожидалась пустота для nil entries")
	}
}

func TestBuildToolCatalog(t *testing.T) {
	if got := BuildToolCatalog(nil); got != "" {
		t.Fatalf("ожидалась пустота, получено %q", got)
	}
	got := BuildToolCatalog([]domain.Tool{{Name: "web_search"}})
	if !strings.Contains(got, "web_search") {
		t.Fatalf("отсутствует имя tool: %q", got)
	}
}

func TestAppendBlock(t *testing.T) {
	if got := AppendBlock("base", "extra"); got != "base\n\nextra" {
		t.Fatalf("получено %q", got)
	}
	if got := AppendBlock("base", ""); got != "base" {
		t.Fatalf("получено %q", got)
	}
}
