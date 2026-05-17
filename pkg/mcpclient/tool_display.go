package mcpclient

import (
	"fmt"
	"strings"
)

type ServerLabelFunc func(serverID int64) string

func ToolProgressDisplayName(normalizedName, rawToolName string, serverLabel ServerLabelFunc) string {
	n := strings.TrimSpace(normalizedName)
	if sid, orig, ok := ParseToolAlias(n); ok {
		label := fmt.Sprintf("MCP #%d", sid)
		if serverLabel != nil {
			if nm := strings.TrimSpace(serverLabel(sid)); nm != "" {
				label = "MCP  " + nm
			}
		}
		orig = strings.TrimSpace(orig)
		if orig != "" {
			return label + "  " + orig
		}
		return label
	}

	if st := strings.TrimSpace(rawToolName); st != "" {
		return st
	}
	return n
}
