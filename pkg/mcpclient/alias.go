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
