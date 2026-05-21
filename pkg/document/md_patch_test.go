package document

import (
	"strings"
	"testing"
)

func TestApplyMarkdownPatchJSONAppendReplace(t *testing.T) {
	const patch = `{"ops":[{"op":"prepend","text":"# "},{"op":"append","text":"\n"},{"op":"replace_substring","old":"x","new":"y"}]}`
	out, err := ApplyMarkdownPatchJSON("Привет x", patch)
	if err != nil {
		t.Fatal(err)
	}

	if out != "# Привет y\n" {
		t.Fatalf("получено %q", out)
	}
}

func TestApplyMarkdownPatchJSONLines(t *testing.T) {
	patch := `{"ops":[
		{"op":"insert_before_line","line":1,"text":"middle"},
		{"op":"delete_line_range","line":0,"count":1},
		{"op":"replace_line_range","line":0,"count":1,"lines":["A","B"]}
	]}`
	out, err := ApplyMarkdownPatchJSON("а\nб\nв", patch)
	if err != nil {
		t.Fatal(err)
	}

	want := "A\nB\nб\nв"
	if out != want {
		t.Fatalf("получено %q, ожидалось %q", out, want)
	}
}

func TestApplyMarkdownPatchJSONAmbiguous(t *testing.T) {
	_, err := ApplyMarkdownPatchJSON("а а", `{"ops":[{"op":"replace_substring","old":"а","new":"б"}]}`)
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}

	if !strings.Contains(err.Error(), "операция") {
		t.Fatalf("err %v", err)
	}
}
