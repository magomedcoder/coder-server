package domain

import (
	"encoding/json"
	"testing"
)

const indexSyncRequestFixture = `{
  "workspace_id": "proj-1",
  "upsert": [
    {
      "id": "chunk-1",
      "path": "src/main.rs",
      "language": "rust",
      "content": "fn main() {}",
      "symbol": "main",
      "symbol_type": "function"
    }
  ],
  "delete": ["chunk-old"],
  "keep_ids": ["chunk-1", "chunk-2"]
}`

func TestIndexSyncRequestFixture(t *testing.T) {
	var req IndexSyncRequest
	if err := json.Unmarshal([]byte(indexSyncRequestFixture), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if req.WorkspaceID != "proj-1" {
		t.Fatalf("workspace_id=%q", req.WorkspaceID)
	}

	if len(req.Upsert) != 1 {
		t.Fatalf("upsert=%d", len(req.Upsert))
	}

	chunk := req.Upsert[0]
	if chunk.Symbol != "main" || chunk.SymbolType != "function" {
		t.Fatalf("symbol fields: %+v", chunk)
	}

	if len(req.Delete) != 1 || req.Delete[0] != "chunk-old" {
		t.Fatalf("delete=%v", req.Delete)
	}

	if len(req.KeepIDs) != 2 {
		t.Fatalf("keep_ids=%v", req.KeepIDs)
	}
}

func TestIndexSyncResponseFixture(t *testing.T) {
	var resp IndexSyncResponse
	if err := json.Unmarshal([]byte(`{"chunks":42}`), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Chunks != 42 {
		t.Fatalf("chunks=%d", resp.Chunks)
	}
}
