package mcpvision

import (
	"testing"
	"time"

	"github.com/magomedcoder/gen/pkg/domain"
)

func TestLastUserMessageHasVisionAttachment(t *testing.T) {
	ts := time.Unix(1, 0)
	msgs := []*domain.Message{
		{
			Role:      domain.MessageRoleUser,
			Content:   "hi",
			CreatedAt: ts,
		},
		{
			Role:      domain.MessageRoleAssistant,
			Content:   "ok",
			CreatedAt: ts,
		},
		{
			Role:              domain.MessageRoleUser,
			Content:           "see",
			CreatedAt:         ts,
			AttachmentName:    "a.png",
			AttachmentMime:    "image/png",
			AttachmentContent: []byte{0x89, 0x50},
		},
	}
	if !LastUserMessageHasVisionAttachment(msgs) {
		t.Fatal("expected true for last user with image bytes")
	}

	msgs2 := []*domain.Message{
		{
			Role:      domain.MessageRoleUser,
			Content:   "only text",
			CreatedAt: ts,
		},
	}
	if LastUserMessageHasVisionAttachment(msgs2) {
		t.Fatal("expected false without image payload")
	}
}

func TestToolAllowedWhenUserHasImage(t *testing.T) {
	list := []string{" 9:read_file ", "1:search"}
	if !ToolAllowedWhenUserHasImage(list, 9, "read_file") {
		t.Fatal("expected allow 9:read_file")
	}

	if ToolAllowedWhenUserHasImage(list, 9, "write_file") {
		t.Fatal("expected deny 9:write_file")
	}

	if !ToolAllowedWhenUserHasImage(nil, 1, "anything") {
		t.Fatal("empty allowlist = allow all")
	}

	if !ToolAllowedWhenUserHasImage([]string{}, 1, "x") {
		t.Fatal("empty slice = allow all")
	}
}
