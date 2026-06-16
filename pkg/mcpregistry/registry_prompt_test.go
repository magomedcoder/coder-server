package mcpregistry

import "testing"

func TestToolsPromptBlockForServersFiltersByID(t *testing.T) {
	reg := &Registry{
		tools: []ToolInfo{
			{
				Alias: "mcp_1_a",
				ServerID: 1,
				ServerName: "wiki",
				Name: "search",
			},
			{
				Alias: "mcp_2_a",
				ServerID: 2,
				ServerName: "docs",
				Name: "read",
			},
		},
	}

	all := reg.ToolsPromptBlockForServers(nil)
	if all == "" || !contains(all, "mcp_1_a") || !contains(all, "mcp_2_a") {
		t.Fatalf("expected both tools in full block, got %q", all)
	}

	filtered := reg.ToolsPromptBlockForServers([]int64{2})
	if contains(filtered, "mcp_1_a") {
		t.Fatalf("server 1 should be filtered out: %q", filtered)
	}

	if !contains(filtered, "mcp_2_a") {
		t.Fatalf("server 2 should remain: %q", filtered)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}

	return -1
}
