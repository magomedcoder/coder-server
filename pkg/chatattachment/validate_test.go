package chatattachment

import (
	"testing"
)

func TestNormalizeAttachmentFileIDsForModel(t *testing.T) {
	t.Run("deduplicate and preserve order", func(t *testing.T) {
		got, err := NormalizeAttachmentFileIDs([]int64{10, 20, 10, 30})
		if err != nil {
			t.Fatalf("неожиданась ошибка: %v", err)
		}
		if len(got) != 3 || got[0] != 10 || got[1] != 20 || got[2] != 30 {
			t.Fatalf("неожиданные normalized ids: %#v", got)
		}
	})

	t.Run("reject non-positive ids", func(t *testing.T) {
		if _, err := NormalizeAttachmentFileIDs([]int64{1, 0}); err == nil {
			t.Fatal("ожидалась ошибка for non-positive attachment id")
		}
	})

	t.Run("reject more than max attachments", func(t *testing.T) {
		ids := []int64{1, 2, 3, 4, 5}
		if _, err := NormalizeAttachmentFileIDs(ids); err == nil {
			t.Fatal("ожидалась ошибка for too many attachments")
		}
	})
}

func TestValidateFileRAGOptions(t *testing.T) {
	t.Run("allow nil options", func(t *testing.T) {
		if err := ValidateFileRAGOptions(nil, nil); err != nil {
			t.Fatalf("неожиданась ошибка: %v", err)
		}
	})

	t.Run("reject file_rag params when use_file_rag=false", func(t *testing.T) {
		err := ValidateFileRAGOptions(&SendMessageFileRAGOptions{
			UseFileRAG: false,
			TopK:       5,
		}, []int64{1})
		if err == nil {
			t.Fatal("ожидалась ошибка валидации")
		}
	})

	t.Run("require exactly one attachment for use_file_rag", func(t *testing.T) {
		err := ValidateFileRAGOptions(&SendMessageFileRAGOptions{
			UseFileRAG: true,
		}, []int64{1, 2})
		if err == nil {
			t.Fatal("ожидалась ошибка валидации")
		}
	})

	t.Run("reject negative top_k", func(t *testing.T) {
		err := ValidateFileRAGOptions(&SendMessageFileRAGOptions{
			UseFileRAG: true,
			TopK:       -1,
		}, []int64{1})
		if err == nil {
			t.Fatal("ожидалась ошибка валидации")
		}
	})
}

func TestNormalizeHydrateParallelism(t *testing.T) {
	if NormalizeHydrateParallelism(0) != 8 {
		t.Fatal("0 -> 8")
	}

	if NormalizeHydrateParallelism(3) != 3 {
		t.Fatal("3")
	}

	if NormalizeHydrateParallelism(100) != 64 {
		t.Fatal("cap 64")
	}
}
