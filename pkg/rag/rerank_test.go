package rag

import (
	"reflect"
	"testing"
)

func TestParseRerankOrder(t *testing.T) {
	got := ParseRerankOrder("3, 1, 2", 3)
	want := []int{2, 0, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("получено %v ожидалось %v", got, want)
	}

	got2 := ParseRerankOrder("1\n2\n3", 3)
	if !reflect.DeepEqual(got2, []int{0, 1, 2}) {
		t.Fatalf("получено %v", got2)
	}
}

func TestTrimPassageForRerank(t *testing.T) {
	s := TrimPassageForRerank("абвгд", 3)
	if s == "" || utf8Count(s) > 4 {
		t.Fatalf("неожиданное значение %q", s)
	}
}

func utf8Count(s string) int {
	n := 0
	for range s {
		n++
	}

	return n
}
