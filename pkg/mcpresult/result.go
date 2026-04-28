package mcpresult

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func Text(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}
}

func TextAndJSON(tool, text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
		StructuredContent: map[string]any{
			"tool":         tool,
			"report_text":  text,
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func TextAndPayload(tool string, payload any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: PrettyJSON(payload),
			},
		},
		StructuredContent: map[string]any{
			"tool":         tool,
			"payload":      payload,
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func PrettyJSON(data any) string {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", data)
	}

	return string(b)
}
