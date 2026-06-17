package contextbudget

import "testing"

func TestTrimChatLinesDropsOldest(t *testing.T) {
	lines := []ChatLine{
		{
			Role:    "user",
			Content: "один " + repeat("a", 400),
		},
		{
			Role:    "assistant",
			Content: "два " + repeat("b", 400),
		},
		{
			Role:    "user",
			Content: "последнее",
		},
	}

	out, changed := TrimChatLines(lines, 120)
	if !changed {
		t.Fatal("ожидалась обрезка истории")
	}

	if len(out) == 0 || out[len(out)-1].Content != "последнее" {
		t.Fatalf("ожидалось сохранить последнее сообщение, получено %#v", out)
	}
}

func TestHistoryTokenBudgetReservesBlocks(t *testing.T) {
	budget := HistoryTokenBudget(8192, 200, 1500, 1024)
	if budget >= 8192 {
		t.Fatalf("бюджет истории должен быть меньше общего: %d", budget)
	}

	if budget < MinContextBudget {
		t.Fatalf("бюджет истории слишком мал: %d", budget)
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}

	return string(out)
}
