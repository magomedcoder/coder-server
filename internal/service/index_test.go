package service

import (
	"testing"

	"github.com/magomedcoder/coder-server/internal/domain"
)

func TestRepoIndexKeywordSearch(t *testing.T) {
	idx := NewRepoIndex(4, nil)
	count, err := idx.Sync(t.Context(), nil, domain.IndexSyncRequest{
		WorkspaceID: "ws1",
		Upsert: []domain.IndexChunk{
			{
				ID:      "1",
				Path:    "a.rs",
				Content: "fn parse_config() {}",
			},
			{
				ID:      "2",
				Path:    "b.rs",
				Content: "struct User { name: String }",
			},
		},
	}, 100)
	if err != nil || count != 2 {
		t.Fatalf("sync не удался: count=%d err=%v", count, err)
	}

	resp, err := idx.Search(t.Context(), nil, domain.SearchRequest{
		WorkspaceID: "ws1",
		Query:       "parse config",
		Limit:       5,
		Mode:        "keyword",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Hits) == 0 || resp.Hits[0].Path != "a.rs" {
		t.Fatalf("неожиданные совпадения: %+v", resp.Hits)
	}
}
