package service

var agentToolNames = map[string]struct{}{
	"list_dir":       {},
	"read_file":      {},
	"glob_search":    {},
	"search_content": {},
	"apply_patch":    {},
	"create_file":    {},
	"run_command":    {},
}

func IsKnownAgentTool(name string) bool {
	_, ok := agentToolNames[name]
	return ok
}

func KnownAgentTools() []string {
	out := make([]string, 0, len(agentToolNames))
	for name := range agentToolNames {
		out = append(out, name)
	}
	return out
}
