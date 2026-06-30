package mcpclient

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func ResultText(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func ResultTextAndJSON(tool, text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		StructuredContent: map[string]any{
			"tool":         tool,
			"report_text":  text,
			"generated_at": time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func ResultTextAndPayload(tool string, payload any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: PrettyJSON(payload)},
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
