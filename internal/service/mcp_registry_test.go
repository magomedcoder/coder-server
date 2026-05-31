package service

import (
	"testing"

	"github.com/magomedcoder/coder-server/internal/config"
	"github.com/magomedcoder/coder-server/internal/domain"
)

func TestRepoIndexGraph(t *testing.T) {
	idx := NewRepoIndex(2, nil)
	_, err := idx.Sync(t.Context(), nil, domain.IndexSyncRequest{
		WorkspaceID: "ws1",
		Upsert: []domain.IndexChunk{
			{
				ID:      "1",
				Path:    "main.rs",
				Content: "mod util;",
				Imports: []string{"util/mod.rs"},
			},
			{
				ID:      "2",
				Path:    "util/mod.rs",
				Content: "pub fn help() {}", Symbol: "help",
			},
		},
	}, 100)
	if err != nil {
		t.Fatal(err)
	}

	graph := idx.Graph("ws1", "main.rs", "")
	if len(graph.Imports) != 1 || graph.Imports[0] != "util/mod.rs" {
		t.Fatalf("неожиданные imports: %+v", graph.Imports)
	}

	sym := idx.Graph("ws1", "", "help")
	if len(sym.ChunkIDs) != 1 || sym.ChunkIDs[0] != "2" {
		t.Fatalf("неожиданный symbol graph: %+v", sym)
	}
}

func TestMCPRegistryDisabled(t *testing.T) {
	reg := NewMCPRegistry(config.MCPConfig{})
	if reg.Enabled() {
		t.Fatal("ожидался отключённый MCP registry")
	}
}
