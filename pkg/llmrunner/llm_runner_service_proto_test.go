package llmrunner

import (
	"testing"
	"time"

	"github.com/magomedcoder/gen-runner/pb/llmrunnerpb"
	"github.com/magomedcoder/gen/pkg/domain"
)

func TestDomainMessagesToProto_preservesVisionAttachmentFields(t *testing.T) {
	img := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00}
	ts := time.Unix(1_700_000_000, 0)
	msgs := []*domain.Message{
		{
			SessionId:         42,
			Role:              domain.MessageRoleUser,
			Content:           "что на изображении",
			CreatedAt:         ts,
			AttachmentName:    "shot.png",
			AttachmentMime:    "image/png",
			AttachmentContent: img,
		},
	}

	out := domainMessagesToProto(msgs)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}

	cm := out[0]
	if cm.GetRole() != string(domain.MessageRoleUser) || cm.GetContent() != "что на изображении" {
		t.Fatalf("role/content: %+v", cm)
	}

	if cm.AttachmentName == nil || *cm.AttachmentName != "shot.png" {
		t.Fatalf("AttachmentName: %+v", cm.AttachmentName)
	}

	if cm.AttachmentMime == nil || *cm.AttachmentMime != "image/png" {
		t.Fatalf("AttachmentMime: %+v", cm.AttachmentMime)
	}

	if string(cm.GetAttachmentContent()) != string(img) {
		t.Fatalf("AttachmentContent len=%d", len(cm.GetAttachmentContent()))
	}
}

func TestCountRunnerVisionAttachments(t *testing.T) {
	if n := countRunnerVisionAttachments([]*domain.Message{
		{
			AttachmentMime:    "image/png",
			AttachmentContent: []byte{1},
		},
		{
			AttachmentMime:    "application/pdf",
			AttachmentContent: []byte{2},
		},
		{
			AttachmentMime:    "",
			AttachmentName:    "x.png",
			AttachmentContent: []byte{3},
		},
	}); n != 2 {
		t.Fatalf("ожидали 2 (png + legacy по имени), got %d", n)
	}
}

func TestDomainMessagesToProto_omitsEmptyAttachmentMime(t *testing.T) {
	msgs := []*domain.Message{
		{
			SessionId:         1,
			Role:              domain.MessageRoleUser,
			Content:           "x",
			CreatedAt:         time.Unix(1, 0),
			AttachmentName:    "a.bin",
			AttachmentMime:    "",
			AttachmentContent: []byte{1, 2, 3},
		},
	}

	out := domainMessagesToProto(msgs)
	if len(out) != 1 {
		t.Fatal(len(out))
	}

	if out[0].AttachmentMime != nil {
		t.Fatalf("ожидали отсутствие optional MIME, got %q", out[0].GetAttachmentMime())
	}
}

func TestDomainMessagesToProto_embedsToolFieldsInContent(t *testing.T) {
	msgs := []*domain.Message{
		func() *domain.Message {
			m := domain.NewMessage(1, "tool out", domain.MessageRoleTool)
			m.ToolCallID = "c1"
			m.ToolName = "mcp_x"
			return m
		}(),
	}

	out := domainMessagesToProto(msgs)
	if len(out) != 1 {
		t.Fatal(len(out))
	}
	c := out[0].GetContent()
	if c == "tool out" || c == "" {
		t.Fatalf("tool meta must be in content: %q", c)
	}
}

func TestApplyRenderedPromptToRequest(t *testing.T) {
	req := &llmrunnerpb.SendMessageRequest{}
	applyRenderedPromptToRequest(req, &domain.GenerationParams{RenderedPrompt: "  <|im_start|>user\nhi"})
	if req.RenderedPrompt == nil || req.GetRenderedPrompt() != "<|im_start|>user\nhi" {
		t.Fatalf("got %v", req.RenderedPrompt)
	}

	applyRenderedPromptToRequest(req, &domain.GenerationParams{RenderedPrompt: "   "})
	if req.RenderedPrompt != nil {
		t.Fatal("whitespace-only must not set field")
	}
}
