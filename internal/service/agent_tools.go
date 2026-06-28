package service

import (
	"strings"

	"github.com/magomedcoder/lmpkg/mcpclient"
)

var agentToolNames = map[string]struct{}{
	"list_dir":       {},
	"read_file":      {},
	"glob_search":    {},
	"search_content": {},
	"apply_patch":    {},
	"create_file":    {},
	"edit_file":      {},
	"rename_file":    {},
	"delete_file":    {},
	"run_command":    {},
}

func IsKnownAgentTool(name string, mcp *MCPRegistry) bool {
	name = strings.TrimSpace(name)
	if _, ok := agentToolNames[name]; ok {
		return true
	}

	if _, _, ok := mcpclient.ParseToolAlias(name); ok {
		if mcp != nil && mcp.ServerExistsByAlias(name) {
			return true
		}
	}

	if mcp != nil && mcp.HasTool(name) {
		return true
	}

	return false
}

func KnownAgentTools() []string {
	out := make([]string, 0, len(agentToolNames))
	for name := range agentToolNames {
		out = append(out, name)
	}
	return out
}
