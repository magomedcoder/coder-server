package mcpclient

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var mcpAliasRe = regexp.MustCompile(`^mcp_(\d+)_h([0-9a-f]+)$`)

func ToolAlias(serverID int64, mcpToolName string) string {
	return fmt.Sprintf("mcp_%d_h%x", serverID, []byte(mcpToolName))
}

func ParseToolAlias(normalized string) (serverID int64, mcpToolName string, ok bool) {
	s := strings.TrimSpace(normalized)
	m := mcpAliasRe.FindStringSubmatch(s)
	if len(m) != 3 {
		return 0, "", false
	}

	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}

	raw, err := hex.DecodeString(m[2])
	if err != nil || len(raw) == 0 {
		return 0, "", false
	}

	return id, string(raw), true
}

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
