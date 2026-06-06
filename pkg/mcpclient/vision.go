package mcpclient

import (
	"fmt"
	"strings"

	"github.com/magomedcoder/gen/pkg/document"
	"github.com/magomedcoder/gen/pkg/domain"
)

func LastUserMessageHasVisionAttachment(msgs []*domain.Message) bool {
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m == nil || m.Role != domain.MessageRoleUser {
			continue
		}

		return MessageCarriesRunnerVisionBytes(m)
	}

	return false
}

func MessageCarriesRunnerVisionBytes(m *domain.Message) bool {
	if m == nil || len(m.AttachmentContent) == 0 {
		return false
	}

	mt := strings.ToLower(strings.TrimSpace(m.AttachmentMime))
	if document.IsAllowedChatImageMIME(mt) || strings.HasPrefix(mt, "image/") {
		return true
	}

	if mt == "" && document.IsImageAttachment(m.AttachmentName) {
		return true
	}

	return false
}

func ToolAllowedWhenUserHasImage(allowlist []string, serverID int64, toolName string) bool {
	if len(allowlist) == 0 {
		return true
	}

	toolName = strings.TrimSpace(toolName)
	if toolName == "" {
		return false
	}

	want := fmt.Sprintf("%d:%s", serverID, toolName)
	for _, e := range allowlist {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}

		if e == want {
			return true
		}
	}

	return false
}
