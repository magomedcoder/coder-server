package toolloop

import (
	"encoding/json"
	"testing"

	"github.com/magomedcoder/coder-server/pkg/domain"
)

func TestTopLevelAllowedPropertyNames_strictFalse(t *testing.T) {
	schema := `{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`
	allowed, strict := TopLevelAllowedPropertyNames(schema)
	if strict || allowed != nil {
		t.Fatalf("ожидалось без additionalProperties=false строгая обрезка не ожидается, strict=%v allowed=%v", strict, allowed)
	}
}

func TestTopLevelAllowedPropertyNames_strictTrue(t *testing.T) {
	schema := `{"type":"object","properties":{"taskId":{"type":"integer"}},"additionalProperties":false}`
	allowed, strict := TopLevelAllowedPropertyNames(schema)
	if !strict || len(allowed) != 1 {
		t.Fatalf("ожидалось strict с 1 ключом, strict=%v len=%d", strict, len(allowed))
	}

	if _, ok := allowed["taskId"]; !ok {
		t.Fatalf("ожидалось taskId в allowed, получено %v", allowed)
	}
}

func TestPruneJSONArgsToSchema_dropsFilter(t *testing.T) {
	schema := `{"type":"object","properties":{"taskId":{"type":"integer"}},"additionalProperties":false}`
	raw := json.RawMessage(`{"filter":{"ID":1001},"taskId":1001}`)
	out, dropped := PruneJSONArgsToSchema(raw, schema, "t")
	if len(dropped) != 1 || dropped[0] != "filter" {
		t.Fatalf("отброшено=%v", dropped)
	}

	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}

	if len(m) != 1 || m["taskId"] != float64(1001) {
		t.Fatalf("неожиданное обрезано object: %v", m)
	}
}

func TestMaybePruneArgsJSON_viaGenParams(t *testing.T) {
	gp := &domain.GenerationParams{
		Tools: []domain.Tool{
			{
				Name:           "mcp_1_habc",
				ParametersJSON: `{"type":"object","properties":{"taskId":{"type":"integer"}},"additionalProperties":false}`,
			},
		},
	}
	raw := json.RawMessage(`{"filter":{},"taskId":42}`)
	out := MaybePruneArgsJSON(gp, "mcp_1_habc", raw)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}

	if len(m) != 1 || m["taskId"] != float64(42) {
		t.Fatalf("неожиданное: %v", m)
	}
}
