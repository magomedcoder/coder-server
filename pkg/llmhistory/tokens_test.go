package llmhistory

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/magomedcoder/gen/pkg/domain"
)

func TestTrimMessagesByApproxTokens_keepsSystemAndTail(t *testing.T) {
	sys := domain.NewMessage(1, "system prompt", domain.MessageRoleSystem)
	u1 := domain.NewMessage(1, strings.Repeat("a", 400), domain.MessageRoleUser)
	a1 := domain.NewMessage(1, strings.Repeat("b", 400), domain.MessageRoleAssistant)
	u2 := domain.NewMessage(1, "last user", domain.MessageRoleUser)

	msgs := []*domain.Message{sys, u1, a1, u2}
	out, trimmed := TrimMessagesByApproxTokens(msgs, 80, 1)
	if !trimmed {
		t.Fatal("ожидалась обрезка")
	}

	if len(out) < 2 {
		t.Fatalf("слишком коротко: %d", len(out))
	}

	if out[0] != sys {
		t.Fatal("system должен остаться первым")
	}

	if out[len(out)-1] != u2 {
		t.Fatal("последнее user должно сохраниться")
	}
}

func TestTrimMessagesByApproxTokens_disabled(t *testing.T) {
	m := domain.NewMessage(1, "x", domain.MessageRoleUser)
	msgs := []*domain.Message{domain.NewMessage(1, "s", domain.MessageRoleSystem), m}
	out, trimmed := TrimMessagesByApproxTokens(msgs, 0, 1)
	if trimmed || len(out) != 2 {
		t.Fatalf("maxTokens=0: trimmed=%v len=%d", trimmed, len(out))
	}
}

func TestTrimMessagesByApproxTokens_systemAndOneUserKeepsInstruction(t *testing.T) {
	sys := domain.NewMessage(1, "system prompt text", domain.MessageRoleSystem)
	u := domain.NewMessage(1, strings.Repeat("ж", 12000), domain.MessageRoleUser)
	msgs := []*domain.Message{sys, u}
	out, trimmed := TrimMessagesByApproxTokens(msgs, 200, 1)
	if !trimmed {
		t.Fatal("ожидалась обрезка при system + одно user (раньше баг: лимит игнорировался)")
	}

	if got := utf8.RuneCountInString(out[1].Content); got != len([]rune(u.Content)) {
		t.Fatalf("последняя инструкция пользователя не должна укорачиваться: runes=%d", got)
	}
}

func TestTrimMessagesByApproxTokensWithDropped_collectsMiddle(t *testing.T) {
	sys := domain.NewMessage(1, "s", domain.MessageRoleSystem)
	u1 := domain.NewMessage(1, strings.Repeat("a", 400), domain.MessageRoleUser)
	a1 := domain.NewMessage(1, strings.Repeat("b", 400), domain.MessageRoleAssistant)
	u2 := domain.NewMessage(1, "tail", domain.MessageRoleUser)
	msgs := []*domain.Message{sys, u1, a1, u2}
	_, trimmed, dropped := TrimMessagesByApproxTokensWithDropped(msgs, 80, 1)
	if !trimmed || len(dropped) < 1 {
		t.Fatalf("trimmed=%v отброшено=%d", trimmed, len(dropped))
	}
}

func TestApproximateMessageTokens_visionImageUsesCappedLenOver16(t *testing.T) {
	img := make([]byte, 80_000)
	m := &domain.Message{
		Role:              domain.MessageRoleUser,
		Content:           "что на фото",
		AttachmentMime:    "image/png",
		AttachmentName:    "x.png",
		AttachmentContent: img,
	}

	got := ApproximateMessageTokens(m)
	wantImgTok := min(8192, max(256, len(img)/16))
	if got < wantImgTok {
		t.Fatalf("ожидалось >= %d токенов (оценка кадра), got=%d", wantImgTok, got)
	}

	pdf := &domain.Message{
		Role:              domain.MessageRoleUser,
		Content:           "что на фото",
		AttachmentMime:    "application/pdf",
		AttachmentName:    "x.pdf",
		AttachmentContent: img,
	}

	gotPDF := ApproximateMessageTokens(pdf)
	if got > gotPDF+64 {
		t.Fatalf("vision не должна сильно превосходить не‑image по байтам: vision=%d pdf=%d", got, gotPDF)
	}
}

func TestTrimMessagesByApproxTokens_dropsVisionUserWithFollowingAssistant(t *testing.T) {
	sys := domain.NewMessage(1, "system", domain.MessageRoleSystem)
	img := make([]byte, 100_000)
	uOld := &domain.Message{
		SessionId:         1,
		Role:              domain.MessageRoleUser,
		Content:           "старый вопрос",
		AttachmentMime:    "image/jpeg",
		AttachmentName:    "old.jpg",
		AttachmentContent: img,
	}
	aOld := domain.NewMessage(1, "ответ по старой картинке", domain.MessageRoleAssistant)
	uTail := domain.NewMessage(1, "новый текст", domain.MessageRoleUser)

	msgs := []*domain.Message{sys, uOld, aOld, uTail}
	out, trimmed, dropped := TrimMessagesByApproxTokensWithDropped(msgs, 120, 1)
	if !trimmed {
		t.Fatal("ожидалась обрезка средней части")
	}

	if len(out) != 2 || out[0] != sys || out[len(out)-1] != uTail {
		t.Fatalf("ожидались system + хвост user: len=%d", len(out))
	}

	if len(dropped) < 2 {
		t.Fatalf("ожидалось снять user+vision и следующего assistant: dropped=%d", len(dropped))
	}

	if dropped[0] != uOld || dropped[1] != aOld {
		t.Fatalf("неожиданное отброшено order: roles %v %v", dropped[0].Role, dropped[1].Role)
	}
}

func TestInjectSummaryAfterSystem(t *testing.T) {
	sys := domain.NewMessage(1, "base", domain.MessageRoleSystem)
	u := domain.NewMessage(1, "u", domain.MessageRoleUser)
	out := InjectSummaryAfterSystem([]*domain.Message{sys, u}, "summary line")
	if len(out) != 2 || out[0] == sys {
		t.Fatal("ожидалась копия system")
	}

	if !strings.Contains(out[0].Content, "summary line") || !strings.Contains(out[0].Content, "base") {
		t.Fatalf("content=%q", out[0].Content)
	}
}
