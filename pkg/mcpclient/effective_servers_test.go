package mcpclient

import (
	"reflect"
	"testing"
)

func TestEffectiveServerIDs_explicitWins(t *testing.T) {
	got := EffectiveServerIDs([]int64{3, 1, 3, -1}, []int64{99})
	want := []int64{1, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestEffectiveServerIDs_fallback(t *testing.T) {
	got := EffectiveServerIDs(nil, []int64{2, 2, 5})
	want := []int64{2, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
