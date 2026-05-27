package delivery

import (
	"context"
	"fmt"
	"strings"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func (h *Handler) enrichContextFromSearch(ctx context.Context, req *domain.ChatRequest) {
	if h == nil || req == nil || req.Search == nil || h.index == nil {
		return
	}

	ws := strings.TrimSpace(req.Search.WorkspaceID)
	if ws == "" {
		return
	}

	query := strings.TrimSpace(req.Search.Query)
	if query == "" && len(req.Messages) > 0 {
		query = strings.TrimSpace(req.Messages[len(req.Messages)-1].Content)
	}
	if query == "" {
		return
	}

	limit := req.Search.Limit
	if limit <= 0 {
		limit = 5
	}

	mode := strings.TrimSpace(req.Search.Mode)
	if mode == "" {
		mode = "hybrid"
	}

	resp, err := h.index.Search(ctx, h.llm, domain.SearchRequest{
		WorkspaceID: ws,
		Query:       query,
		Limit:       limit,
		Mode:        mode,
	})
	if err != nil || len(resp.Hits) == 0 {
		return
	}

	if req.Context == nil {
		req.Context = &domain.ChatContext{}
	}

	for _, hit := range resp.Hits {
		snippet := domain.ContextSnippet{
			Path:     hit.Path,
			Language: hit.Language,
			Content:  hit.Snippet,
			Source:   "codebase",
		}
		if hit.Symbol != "" {
			snippet.Content = fmt.Sprintf("// %s\n%s", hit.Symbol, hit.Snippet)
		}
		req.Context.Snippets = append(req.Context.Snippets, snippet)
	}
}
