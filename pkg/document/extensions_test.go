package document

import (
	"context"
	"strings"
	"testing"
)

func TestIsSupportedOrPlainExtra(t *testing.T) {
	extra := []string{".adoc", "textile"}
	if !IsSupportedOrPlainExtra("x.md", extra) {
		t.Fatal("md must stay supported")
	}

	if !IsSupportedOrPlainExtra("notes.adoc", extra) {
		t.Fatal("expected .adoc in extra")
	}

	if !IsSupportedOrPlainExtra("x.TEXTILE", extra) {
		t.Fatal("expected .textile case-insensitive")
	}

	if IsSupportedOrPlainExtra("x.unknownext", extra) {
		t.Fatal("unknown must fail")
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
		t.Fatalf("bounds: %v", bounds)
	}

	if got == "" || !strings.Contains(got, "wikiext_marker_771") {
		t.Fatalf("text: %q", got)
	}
}
