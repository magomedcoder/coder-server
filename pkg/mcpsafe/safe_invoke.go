package mcpsafe

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func SafeToolInvoke(origin, tool string, fn func() (*mcp.CallToolResult, any, error)) (res *mcp.CallToolResult, meta any, err error) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		origin = "mcp-server"
	}

	tool = strings.TrimSpace(tool)
	if tool == "" {
		tool = "unknown_tool"
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("%s: tool=%q panic recovered: %v\n%s", origin, tool, r, debug.Stack())
			var out mcp.CallToolResult
			out.SetError(fmt.Errorf("%s: внутренняя ошибка в инструменте %q: %v", origin, tool, r))
			res = &out
			meta = nil
			err = nil
		}
	}()

	return fn()
}
