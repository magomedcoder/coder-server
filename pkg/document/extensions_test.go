package document

import (
	"context"
	"strings"
	"testing"
)

func TestIsSupportedOrPlainExtra(t *testing.T) {
	extra := []string{".adoc", "textile"}
	if !IsSupportedOrPlainExtra("x.md", extra) {
		t.Fatal("md должно оставаться supported")
	}

	if !IsSupportedOrPlainExtra("notes.adoc", extra) {
		t.Fatal("ожидалось .adoc в extra")
	}

	if !IsSupportedOrPlainExtra("x.TEXTILE", extra) {
		t.Fatal("ожидалось .textile без учёта регистра")
	}

	if IsSupportedOrPlainExtra("x.unknownext", extra) {
		t.Fatal("unknown должно завершаться ошибкой")
	}
}

func TestExtractTextForRAGOrPlainExtra(t *testing.T) {
	ctx := context.Background()
	body := []byte("# Title\n\nextra plain wikiext_marker_771\n")
	got, bounds, err := ExtractTextForRAGOrPlainExtra(ctx, "doc.wikiext", body, []string{"wikiext"})
	if err != nil {
		t.Fatal(err)
	}

	if len(bounds) != 0 {
		t.Fatalf("границы: %v", bounds)
	}

	if got == "" || !strings.Contains(got, "wikiext_marker_771") {
		t.Fatalf("text: %q", got)
	}
}
