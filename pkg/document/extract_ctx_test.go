package document

import (
	"context"
	"testing"
)

func TestExtractTextForRAGContextAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := ExtractTextForRAGContext(ctx, "x.txt", []byte("hello"))
	if err == nil {
		t.Fatal("expected ctx error")
	}

	if err != context.Canceled {
		t.Fatalf("expected canceled, got %v", err)
	}
}
