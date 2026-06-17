package delivery

import (
	"log"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func logReq(requestID, format string, args ...any) {
	if requestID != "" {
		log.Printf("request_id=%s "+format, append([]any{requestID}, args...)...)
		return
	}

	log.Printf(format, args...)
}

func logPreview(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if maxRunes <= 0 {
		maxRunes = 80
	}

	rs := []rune(s)
	if len(rs) <= maxRunes {
		return s
	}

	return string(rs[:maxRunes]) + "..."
}

func chatWorkspaceLabel(req domain.ChatRequest) string {
	if req.Context != nil && req.Context.Workspace != nil {
		ws := req.Context.Workspace
		if n := strings.TrimSpace(ws.Name); n != "" {
			return n
		}

		if r := strings.TrimSpace(ws.Root); r != "" {
			return r
		}
	}

	if req.Search != nil {
		if ws := strings.TrimSpace(req.Search.WorkspaceID); ws != "" {
			return ws
		}
	}

	return "-"
}

func chatSnippetCount(req domain.ChatRequest) int {
	if req.Context == nil {
		return 0
	}

	return len(req.Context.Snippets)
}

func chatEditorPath(req domain.ChatRequest) string {
	if req.Editor == nil {
		return "-"
	}

	if p := strings.TrimSpace(req.Editor.Path); p != "" {
		return p
	}

	return "-"
}

func sessionLabel(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "-"
	}

	return id
}
